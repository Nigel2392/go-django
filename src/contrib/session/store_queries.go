package session

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/query_errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

var (
	_ migrator.IndexDefiner = (*Session)(nil)
	_ attrs.Definer         = (*Session)(nil)
)

type Session struct {
	//	token CHAR(43) PRIMARY KEY,
	//  data BLOB NOT NULL,
	//  expiry TIMESTAMP(6) NOT NULL
	Token  drivers.Char
	Data   drivers.Bytes
	Expiry int64 // Use int64 to store Unix timestamp in nanoseconds
}

func (s *Session) FieldDefs() attrs.Definitions {
	return attrs.Define(s,
		attrs.Unbound("Token", &attrs.FieldConfig{
			Primary:   true,
			Null:      false,
			Column:    "token",
			MaxLength: 43,
		}),
		attrs.Unbound("Data", &attrs.FieldConfig{
			Null:   false,
			Column: "data",
		}),
		attrs.Unbound("Expiry", &attrs.FieldConfig{
			Null:   false,
			Column: "expiry",
		}),
	).WithTableName("sessions")
}

func (s *Session) DatabaseIndexes() []migrator.Index {
	return []migrator.Index{
		{
			Identifier: "sessions_expiry_idx",
			Fields:     []string{"Expiry"},
		},
		{
			Identifier: "sessions_token_idx",
			Fields:     []string{"Token"},
			Unique:     true,
		},
	}
}

var (
	cleanupMu = &sync.Mutex{}
)

type QueryStore struct {
	db          drivers.Database
	stopCleanup chan bool
}

// NewQueryStore returns a new QueryStore instance, with a background cleanup goroutine
// that runs every 5 minutes to remove expired session data.
func NewQueryStore(db drivers.Database) *QueryStore {
	return NewQueryStoreWithCleanupInterval(db, 5*time.Minute)
}

// NewQueryStoreWithCleanupInterval returns a new QueryStore instance. The cleanupInterval
// parameter controls how frequently expired session data is removed by the
// background cleanup goroutine. Setting it to 0 prevents the cleanup goroutine
// from running (i.e. expired sessions will not be removed).
func NewQueryStoreWithCleanupInterval(db drivers.Database, cleanupInterval time.Duration) *QueryStore {
	p := &QueryStore{db: db}
	if cleanupInterval > 0 {
		go p.startCleanup(cleanupInterval)
	}
	return p
}

// Find returns the data for a given session token from the QueryStore instance.
// If the session token is not found or is expired, the returned exists flag will
// be set to false.
func (p *QueryStore) Find(token string) ([]byte, bool, error) {
	var row, err = queries.GetQuerySet(&Session{}).
		Filter("Token", token).
		Filter("Expiry__gt", time.Now().UTC().UnixNano()).
		Get()
	if err != nil && errors.Is(err, query_errors.ErrNoRows) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	return row.Object.Data, true, nil
}

// Commit adds a session token and data to the QueryStore instance with the
// given expiry time. If the session token already exists, then the data and expiry
// time are updated.
func (p *QueryStore) Commit(token string, b []byte, expiry time.Time) error {

	var session = &Session{
		Token:  drivers.Char(token),
		Data:   drivers.Bytes(b),
		Expiry: expiry.UTC().UnixNano(),
	}

	var querySet = queries.GetQuerySet(session).Filter("Token", token)
	var transaction, err = querySet.GetOrCreateTransaction()
	if err != nil {
		return err
	}
	defer transaction.Rollback(context.Background())

	obj, created, err := querySet.GetOrCreate(session)
	if err != nil {
		return err
	}

	if created {
		return transaction.Commit(context.Background())

	}

	// Update existing session data and expiry
	obj.Data = session.Data
	obj.Expiry = session.Expiry
	_, err = querySet.Update(obj)
	if err != nil {
		return err
	}

	return transaction.Commit(context.Background())

}

// Delete removes a session token and corresponding data from the QueryStore
// instance.
func (p *QueryStore) Delete(token string) error {
	var _, err = queries.GetQuerySet(&Session{}).Filter("Token", token).Delete()
	return err
}

// All returns a map containing the token and data for all active (i.e.
// not expired) sessions in the QueryStore instance.
func (p *QueryStore) All() (map[string][]byte, error) {
	var rows, err = queries.GetQuerySet(&Session{}).
		Select("Token", "Data").
		Filter("Expiry__gt", time.Now().UTC().UnixNano()).
		All()
	if err != nil {
		return nil, err
	}

	var sessions = make(map[string][]byte)
	for _, row := range rows {
		sessions[string(row.Object.Token)] = row.Object.Data
	}

	return sessions, nil
}

func (p *QueryStore) startCleanup(interval time.Duration) {
	p.stopCleanup = make(chan bool)
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			cleanupMu.Lock()
			err := p.deleteExpired()
			if err != nil {
				log.Println(err)
			}
			cleanupMu.Unlock()
		case <-p.stopCleanup:
			ticker.Stop()
			return
		}
	}
}

// StopCleanup terminates the background cleanup goroutine for the QueryStore
// instance. It's rare to terminate this; generally QueryStore instances and
// their cleanup goroutines are intended to be long-lived and run for the lifetime
// of your application.
//
// There may be occasions though when your use of the QueryStore is transient.
// An example is creating a new QueryStore instance in a test function. In this
// scenario, the cleanup goroutine (which will run forever) will prevent the
// QueryStore object from being garbage collected even after the test function
// has finished. You can prevent this by manually calling StopCleanup.
func (p *QueryStore) StopCleanup() {
	if p.stopCleanup != nil {
		p.stopCleanup <- true
	}
}

func (p *QueryStore) deleteExpired() error {
	var _, err = queries.GetQuerySet(&Session{}).Filter("Expiry__lt", time.Now().UTC().UnixNano()).Delete()
	return err
}
