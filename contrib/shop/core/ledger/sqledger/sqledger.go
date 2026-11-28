package sqledger

import (
	"context"
	"database/sql"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"time"
	"uuid"

	"github.com/Nigel2392/go-django/contrib/shop/core/ledger"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
)

const leaseDuration = time.Minute

type Lease struct {
	id           drivers.UUID
	data         string
	expiry       time.Time
	tries        int
	status       ledger.Status
	started      time.Time
	updated      time.Time
	finished     time.Time
	errorCode    string
	errorMessage string
}

func (l *Lease) ID() uuid.UUID {
	return uuid.UUID(l.id)
}

func (l *Lease) ExpiresAt() time.Time {
	return l.expiry
}

func (l *Lease) Tries() int {
	return l.tries
}

func (l *Lease) Status() ledger.Status {
	return l.status
}

func (l *Lease) StartedAt() time.Time {
	return l.started
}

func (l *Lease) UpdatedAt() time.Time {
	return l.updated
}

func (l *Lease) FinishedAt() (time.Time, bool) {
	return l.finished, !l.finished.IsZero()
}

func (l *Lease) Failure() (code, message string) {
	return l.errorCode, l.errorMessage
}

func (l *Lease) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, l,
		fattrs.Field(l, "id", &l.id, fattrs.ConfPrimary),
		fattrs.Field(l, "data", &l.data, fattrs.ConfNullBlank),
		fattrs.Field(l, "expiry", &l.expiry, fattrs.ConfZero),
		fattrs.Field(l, "tries", &l.tries, fattrs.ConfZero),
		fattrs.Field(l, "status", &l.status, fattrs.ConfZero),
		fattrs.Field(l, "started", &l.started, fattrs.ConfNullBlank),
		fattrs.Field(l, "updated", &l.updated, fattrs.ConfNullBlank),
		fattrs.Field(l, "finished", &l.finished, fattrs.ConfNullBlank),
		fattrs.Field(l, "errorCode", &l.errorCode, fattrs.ConfNullBlank),
		fattrs.Field(l, "errorMessage", &l.errorMessage, fattrs.ConfNullBlank),
	)
}

type Ledger[TASK any] struct {
	Work        func(context.Context, TASK, ledger.TaskState) (err error)
	EncoderOpts []jsontext.Options
	DecoderOpts []jsontext.Options
}

func (l *Ledger[T]) Start(ctx context.Context, taskData T) (ledger.Task, error) {

	data, err := json.Marshal(taskData, l.EncoderOpts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to encode task data %T", taskData)
	}

	id := drivers.UUID(uuid.NewV7())
	lease := new(Lease{
		id:     id,
		status: ledger.PENDING,
		data:   string(data),
	})

	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "cannot start ledger transaction")
	}

	defer tx.Rollback(ctx)

	lease, err = queries.
		GetQuerySet(&Lease{}).
		WithContext(ctx).
		Create(lease)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start lease %s", id.String())
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.Wrapf(err, "could not commit transaction for lease %s", id.String())
	}

	return lease, nil
}

// DoWork handles one pending task or takes over one expired claim. A successful
// Work call and the completed task state commit in the same transaction.
func (l *Ledger[T]) DoWork(ctx context.Context) error {
	if l.Work == nil || queries.IsTransactionContext(ctx) || !queries.IsCommitContext(ctx) {
		return errors.ValueError.Wrap(
			"ledger worker requires a work function and a writable context outside a transaction",
		)
	}

	now := time.Now().UTC()
	due := expr.Or[expr.Expression](
		expr.Q("status", ledger.PENDING),
		expr.And(
			expr.Q("status", ledger.RUNNING),
			expr.Q("expiry__lte", now),
		),
	)

	row, err := queries.
		GetQuerySet(&Lease{}).
		WithContext(ctx).
		Filter(due).
		OrderBy("id").
		First()
	if errors.Is(err, errors.NoRows) {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "find available ledger task")
	}

	task := row.Object
	expiry := now.Add(leaseDuration)
	started := task.started
	if started.IsZero() {
		started = now
	}

	// Claim the task before opening the work transaction so a crashed worker
	// leaves an expiring claim that another worker can take over.
	count, err := queries.
		GetQuerySet(&Lease{}).
		WithContext(ctx).
		Filter("id", task.id).
		Filter("tries", task.tries).
		Filter(due).
		BulkUpdate(
			expr.As("status", expr.V(ledger.RUNNING)),
			expr.As("tries", expr.Logical("tries").ADD(expr.V(1))),
			expr.As("expiry", expr.V(expiry)),
			expr.As("started", expr.V(started)),
			expr.As("updated", expr.V(now)),
		)
	if err != nil || count == 0 {
		return errors.Wrap(err, "claim ledger task")
	}

	task.status = ledger.RUNNING
	task.tries++
	task.expiry = expiry
	task.started = started
	task.updated = now

	workCtx, cancel := context.WithDeadline(ctx, expiry)
	defer cancel()

	workCtx, tx, err := queries.StartTransaction(workCtx)
	if err != nil {
		return errors.Wrap(err, "start ledger work transaction")
	}

	rollbackCtx := context.WithoutCancel(workCtx)
	defer tx.Rollback(rollbackCtx)

	var (
		code, message string

		status   = ledger.COMPLETED
		writeCtx = workCtx
	)

	workErr := l.workItem(workCtx, task)
	if workErr != nil {
		if err := tx.Rollback(rollbackCtx); err != nil && !errors.Is(err, sql.ErrTxDone) {
			return errors.Wrap(err, "roll back failed ledger work")
		}

		// Record the callback error after its writes have been rolled back.
		status = ledger.FAILED
		code = "work_failed"
		message = workErr.Error()
		writeCtx = ctx
	}

	now = time.Now().UTC()
	count, err = queries.
		GetQuerySet(&Lease{}).
		WithContext(writeCtx).
		Filter("id", task.id).
		Filter("tries", task.tries).
		Filter("status", ledger.RUNNING).
		Filter("expiry__gt", now).
		BulkUpdate(
			expr.As("status", expr.V(status)),
			expr.As("updated", expr.V(now)),
			expr.As("finished", expr.V(now)),
			expr.As("errorCode", expr.V(code)),
			expr.As("errorMessage", expr.V(message)),
		)
	if err != nil {
		return errors.Wrap(err, "record ledger task outcome")
	}
	if workErr != nil || count == 0 {
		return nil
	}

	return errors.Wrap(tx.Commit(workCtx), "commit ledger work")
}

func (l *Ledger[T]) workItem(ctx context.Context, task *Lease) error {

	var out T
	var err = json.Unmarshal([]byte(task.data), &out, l.DecoderOpts...)
	if err != nil {
		return errors.Wrapf(err, "failed to decode data into %T", out)
	}

	err = l.Work(ctx, out, task)
	if err != nil {
		return errors.Wrapf(err, "failed to execute job for task %s", task.id.String())
	}

	return nil
}

func (l *Ledger[T]) Progress(ctx context.Context, taskId uuid.UUID) (ledger.TaskState, error) {
	leaseRow, err := queries.
		GetQuerySet(&Lease{}).
		WithContext(ctx).
		Filter("id", drivers.UUID(taskId)).
		Get()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve task state for lease %s", taskId)
	}

	return leaseRow.Object, nil
}
