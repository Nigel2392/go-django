package quest

import (
	"context"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	sqle "github.com/dolthub/go-mysql-server"
	"github.com/dolthub/go-mysql-server/memory"
	"github.com/dolthub/go-mysql-server/server"
	doltsql "github.com/dolthub/go-mysql-server/sql"
)

type QuestDatabase interface {
	DB() (drivers.Database, error)
	Start() error
	Stop() error
}

type DatabaseConfig struct {
	DBName       string
	ServerConfig server.Config
	ProviderOpts []memory.ProviderOption
}

type inMemoryDatabase struct {
	db     doltsql.Database
	serve  *server.Server
	config *DatabaseConfig
}

func (d *inMemoryDatabase) DB() (drivers.Database, error) {
	var dsn = fmt.Sprintf(
		"root@tcp(%s)/%s?parseTime=true&multiStatements=true&interpolateParams=true",
		d.config.ServerConfig.Address, d.config.DBName,
	)
	var db, err = drivers.Open(context.Background(), "mysql", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (d *inMemoryDatabase) Start() error {
	if err := d.serve.Start(); err != nil {
		return err
	}
	return nil
}

func (d *inMemoryDatabase) Stop() error {
	if err := d.serve.Close(); err != nil {
		return err
	}
	return nil
}

func MySQLDatabase(cnf DatabaseConfig) (QuestDatabase, error) {
	if cnf.DBName == "" {
		return nil, fmt.Errorf("database name cannot be empty")
	}

	if cnf.ServerConfig.Protocol == "" {
		cnf.ServerConfig.Protocol = "tcp"
	}

	if cnf.ServerConfig.Address == "" {
		cnf.ServerConfig.Address = "127.0.0.1:13306"
	}

	var db = memory.NewDatabase(cnf.DBName)
	db.BaseDatabase.EnablePrimaryKeyIndexes()

	var provider = memory.NewDBProviderWithOpts(append(
		[]memory.ProviderOption{
			memory.WithDbsOption([]doltsql.Database{db}),
		},
		cnf.ProviderOpts...,
	)...)
	var engine = sqle.NewDefault(provider)
	var server, err = server.NewServer(
		cnf.ServerConfig, engine, doltsql.NewContext, memory.NewSessionBuilder(provider.(*memory.DbProvider)), nil,
	)
	if err != nil {
		return nil, err
	}

	return &inMemoryDatabase{
		db:     db,
		serve:  server,
		config: &cnf,
	}, nil
}
