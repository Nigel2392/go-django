//go:build !sqlite && !postgres && !mariadb && !mysql && mysql_local

package testdb

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/quest"
	"github.com/dolthub/go-mysql-server/server"
)

const ENGINE = "mysql_local"

func getFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "127.0.0.1:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}

func open() (which string, db drivers.Database) {

	var port, err = getFreePort()
	if err != nil {
		panic(fmt.Errorf("failed to get free port: %w", err))
	}

	questDb, err := quest.MySQLDatabase(quest.DatabaseConfig{
		DBName: "django-test",
		ServerConfig: server.Config{
			Address: fmt.Sprintf("127.0.0.1:%d", port),
		},
	})
	if err != nil {
		panic(err)
	}

	go func() {
		if err := questDb.Start(); err != nil {
			panic(err)
		}
	}()

	go func() {
		// Listen for exit signal
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		if err := questDb.Stop(); err != nil {
			fmt.Printf("Error stopping database: %v\n", err)
		}
	}()

	db, err = questDb.DB()
	if err != nil {
		panic(err)
	}

	for i := 0; i < retries; i++ {
		//  Wait for the database to be ready
		if err := db.Ping(context.Background()); err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}

	return ENGINE, db
}
