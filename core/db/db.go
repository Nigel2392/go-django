package db

import (
	"strconv"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Database key type.
// This can be used to fetch a database connection from the pool.
type DATABASE_KEY string

const (
	// Default database key.
	DEFAULT_DATABASE_KEY DATABASE_KEY = "default"
)

type PoolItem[T any] interface {
	Key() DATABASE_KEY
	DB() T
	Models() []interface{}
	Register(model ...interface{}) error
	AutoMigrate() error
}

// A pool of database connections.
type Pool[T any] interface {
	Add(DB_KEY DATABASE_KEY, DB T) error
	Get(DB_KEY DATABASE_KEY) (PoolItem[T], error)
	ByModel(model interface{}) (PoolItem[T], error)
	Delete(DB_KEY DATABASE_KEY)
	AutoMigrate() error
	Register(DB_KEY DATABASE_KEY, model ...interface{}) error
	GetDefaultDB() PoolItem[T]
}

type Manager struct {
	DEFAULT_DATABASE string
	DB_NAME          string
	DB_USER          string
	DB_PASS          string
	DB_HOST          string
	DB_PORT          int
	DB_SSLMODE       string
	Config           *gorm.Config
	DB               *gorm.DB
	pool             Pool[*gorm.DB]
}

func (d *Manager) Pool() Pool[*gorm.DB] {
	if d.pool == nil {
		d.pool = NewPool(d.DB)
	}
	return d.pool
}

func (d *Manager) AutoMigrate(models ...interface{}) {
	d.DB.AutoMigrate(models...)
}

func (d *Manager) createSQLDSN() string {
	var host = d.DB_HOST
	var port = d.DB_PORT
	var user = d.DB_USER
	var password = d.DB_PASS
	var database = d.DB_NAME
	var sslmode = d.DB_SSLMODE
	if user == "" {
		if sslmode != "" {
			return host + ":" + strconv.Itoa(port) + "/" + database + "?sslmode=" + sslmode + "&charset=utf8&parseTime=True&loc=Local"
		}
		return host + ":" + strconv.Itoa(port) + "/" + database + "?charset=utf8&parseTime=True&loc=Local"
	}
	if sslmode != "" {
		return user + ":" + password + "@tcp(" + host + ":" + strconv.Itoa(port) + ")/" + database + "?sslmode=" + sslmode + "&charset=utf8&parseTime=True&loc=Local"
	}
	return user + ":" + password + "@tcp(" + host + ":" + strconv.Itoa(port) + ")/" + database + "?charset=utf8&parseTime=True&loc=Local"
}

func (d *Manager) Init() *gorm.DB {
	var configuration = &gorm.Config{}
	if d.Config != nil {
		configuration = d.Config
	}

	var db *gorm.DB
	var err error

	switch strings.ToLower(d.DEFAULT_DATABASE) {
	case "mysql", "mariadb":
		db, err = gorm.Open(mysql.Open(d.createSQLDSN()), configuration)
	case "sqlite", "sqlite3":
		db, err = gorm.Open(sqlite.Open(d.DB_NAME), configuration)
	//	case "postgres", "postgresql":
	//		db, err = gorm.Open(postgres.Open(d.createSQLDSN()), configuration)
	//	case "mssql", "sqlserver":
	//		db, err = gorm.Open(sqlserver.Open(d.createSQLDSN()), configuration)
	default:
		panic("Unknown database type")
	}
	if err != nil {
		panic(err)
	}

	if db != nil {
		d.DB = db
		return db
	}

	db, err = gorm.Open(sqlite.Open("default.db"), configuration)
	if err != nil {
		panic(err)
	}
	d.DB = db
	return db
}
