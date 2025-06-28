package session_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/djester/testdb"
	_ "github.com/go-sql-driver/mysql"
)

func createTable(db drivers.Database) error {
	var schemaEditor, err = migrator.GetSchemaEditor(db.Driver())
	if err != nil {
		return fmt.Errorf("failed to get schema editor: %w", err)
	}

	var table = migrator.NewModelTable(&session.Session{})
	if err := schemaEditor.CreateTable(table, true); err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	for _, index := range table.Indexes() {
		if err := schemaEditor.AddIndex(table, index, true); err != nil {
			return fmt.Errorf("failed to create index %s: %w", index.Name(), err)
		}
	}
	return nil
}

var db drivers.Database

func init() {
	attrs.RegisterModel(&session.Session{})

	_, db = testdb.Open()

	if django.Global == nil {
		django.App(django.Configure(map[string]interface{}{
			django.APPVAR_DATABASE: db,
		}))

		logger.Setup(&logger.Logger{
			Level:       logger.DBG,
			WrapPrefix:  logger.ColoredLogWrapper,
			OutputDebug: os.Stdout,
			OutputInfo:  os.Stdout,
			OutputWarn:  os.Stdout,
			OutputError: os.Stderr,
		})
	}

	if err := createTable(db); err != nil {
		panic(fmt.Errorf("failed to create sessions table: %w", err))
	}
}

func getNowFromDB() time.Time {
	var row = queries.GetQuerySet(&session.Session{}).Annotate("now", expr.NOW()).Row(
		"SELECT ![now]",
	)
	var now time.Time
	if err := row.Scan(&now); err != nil {
		panic(fmt.Errorf("failed to get current time from database: %w", err))
	}
	return now
}

func TestFind(t *testing.T) {
	m := session.NewQueryStoreWithCleanupInterval(db, 0)

	var _, err = queries.GetQuerySet(&session.Session{}).Create(&session.Session{
		Token:  "session_token",
		Data:   []byte("encoded_data"),
		Expiry: time.Now().Add(time.Minute).UTC().UnixNano(),
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	b, found, err := m.Find("session_token")
	if err != nil {
		t.Fatal(err)
	}
	if found != true {
		t.Logf("current time from DB: %s", getNowFromDB())
		t.Fatalf("got %v: expected %v", found, true)
	}
	if bytes.Equal(b, []byte("encoded_data")) == false {
		t.Fatalf("got %v: expected %v", b, []byte("encoded_data"))
	}
}

func TestFindMissing(t *testing.T) {
	m := session.NewQueryStoreWithCleanupInterval(db, 0)

	_, found, err := m.Find("missing_session_token")
	if err != nil {
		t.Fatalf("got %v: expected %v", err, nil)
	}
	if found != false {
		t.Fatalf("got %v: expected %v", found, false)
	}
}

func TestSaveNew(t *testing.T) {
	m := session.NewQueryStoreWithCleanupInterval(db, 0)

	var err = m.Commit("session_token", []byte("encoded_data"), time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	row := db.QueryRowContext(context.Background(), "SELECT data FROM sessions WHERE token = 'session_token'")
	var data []byte
	err = row.Scan(&data)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(data, []byte("encoded_data")) == false {
		t.Fatalf("got %v: expected %v", data, []byte("encoded_data"))
	}
}

func TestSaveUpdated(t *testing.T) {
	m := session.NewQueryStoreWithCleanupInterval(db, 0)

	var err = m.Commit("session_token", []byte("new_encoded_data"), time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	row := db.QueryRowContext(context.Background(), "SELECT data FROM sessions WHERE token = 'session_token'")
	var data []byte
	err = row.Scan(&data)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(data, []byte("new_encoded_data")) == false {
		t.Fatalf("got %v: expected %v", data, []byte("new_encoded_data"))
	}
}

func TestExpiry(t *testing.T) {
	m := session.NewQueryStoreWithCleanupInterval(db, 200*time.Millisecond)

	var err = m.Commit("expired_token", []byte("encoded_data"), time.Now().Add(400*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}

	_, found, _ := m.Find("expired_token")
	if found != true {
		t.Fatalf("got %v: expected %v", found, true)
	}

	time.Sleep(600 * time.Millisecond)

	obj, found, err := m.Find("expired_token")
	if found != false {

		if obj != nil {
			t.Logf("object data: %v", obj)
		}

		t.Fatalf("got %v: expected %v (%v)", found, false, nil)
	}
}

func TestDelete(t *testing.T) {
	m := session.NewQueryStoreWithCleanupInterval(db, 0)

	var err = m.Delete("session_token")
	if err != nil {
		t.Fatal(err)
	}

	row := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM sessions WHERE token = 'session_token'")
	var count int
	err = row.Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("got %d: expected %d", count, 0)
	}
}

func TestCleanup(t *testing.T) {
	m := session.NewQueryStoreWithCleanupInterval(db, 200*time.Millisecond)

	defer m.StopCleanup()

	var err = m.Commit("cleanup_token", []byte("encoded_data"), time.Now().Add(400*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}

	row := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM sessions WHERE token = 'cleanup_token'")
	var count int
	err = row.Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("got %d: expected %d", count, 1)
	}

	time.Sleep(650 * time.Millisecond)

	row = db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM sessions WHERE token = 'cleanup_token'")
	err = row.Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("got %d: expected %d", count, 0)
	}
}

func TestStopNilCleanup(t *testing.T) {
	m := session.NewQueryStoreWithCleanupInterval(db, 0)

	time.Sleep(100 * time.Millisecond)
	// A send to a nil channel will block forever
	m.StopCleanup()
}
