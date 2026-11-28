package ledger

import (
	"context"
	"uuid"

	"github.com/pkg/errors"
)

var (
	ErrLedgerNotExists   = errors.New("ledger does not exist")
	ErrInvalidLedgerType = errors.New("invalid ledger type")

	Ledgers = make(LedgerRegistry)
)

type LedgerRegistry map[uuid.UUID]func(context.Context) (any, error)

func (l LedgerRegistry) Add[TASK any](newLedger func(uuid.UUID, context.Context) (Ledger[TASK], error)) (ledgerId uuid.UUID) {
	if newLedger == nil {
		panic("newLedger function cannot be nil")
	}

	ledgerId = uuid.NewV7()
	l[ledgerId] = func(ctx context.Context) (any, error) { return newLedger(ledgerId, ctx) }
	return ledgerId
}

func (l LedgerRegistry) get(ctx context.Context, ledgerId uuid.UUID) (any, error) {
	newLedger, ok := l[ledgerId]
	if !ok {
		return nil, ErrLedgerNotExists
	}

	iledger, err := newLedger(ctx)
	if err != nil {
		return nil, errors.Wrapf(
			err, "could not build ledger for %s", ledgerId.String(),
		)
	}

	return iledger, nil
}

func (l LedgerRegistry) Get[TASK any](ctx context.Context, ledgerId uuid.UUID) (Ledger[TASK], error) {
	iledger, err := l.get(ctx, ledgerId)
	if err != nil {
		return nil, err
	}

	ledger, ok := iledger.(Ledger[TASK])
	if !ok {
		return nil, ErrInvalidLedgerType
	}

	return ledger, nil
}

func (l LedgerRegistry) Start[TASK any](ctx context.Context, ledgerId uuid.UUID, taskData TASK) (Task, error) {
	ledger, err := l.Get[TASK](ctx, ledgerId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to start task")
	}

	return ledger.Start(ctx, taskData)
}

var _ (progressor) = (Ledger[any])(nil)

type progressor interface {
	Progress(ctx context.Context, taskId uuid.UUID) (TaskState, error)
}

func (l LedgerRegistry) Progress(ctx context.Context, ledgerId uuid.UUID, taskId uuid.UUID) (TaskState, error) {
	iledger, err := l.get(ctx, ledgerId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve task progress")
	}

	p, ok := iledger.(progressor)
	if !ok {
		return nil, errors.Wrap(ErrInvalidLedgerType, "failed to retrieve task progress")
	}

	return p.Progress(ctx, taskId)
}
