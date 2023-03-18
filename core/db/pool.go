package db

import (
	"errors"
	"sync"

	"github.com/Nigel2392/go-django/core/models"
	"gorm.io/gorm"
)

type databasePoolItem struct {
	key    DATABASE_KEY
	db     *gorm.DB
	models []interface{}
}

func (m *databasePoolItem) DB() *gorm.DB {
	return m.db
}

func (m *databasePoolItem) Models() []interface{} {
	return m.models
}

func (m *databasePoolItem) Key() DATABASE_KEY {
	return m.key
}

type defaultPool struct {
	mu        sync.RWMutex
	databases map[DATABASE_KEY]*databasePoolItem
}

func NewPool(defaultDB *gorm.DB) Pool[*gorm.DB] {
	var p = &defaultPool{
		databases: make(map[DATABASE_KEY]*databasePoolItem),
	}
	p.store(DEFAULT_DATABASE_KEY, &databasePoolItem{
		db:     defaultDB,
		models: make([]interface{}, 0),
	})
	return p
}

func (m *defaultPool) load(key DATABASE_KEY) (value *databasePoolItem, ok bool) {
	m.mu.RLock()
	value, ok = m.databases[key]
	m.mu.RUnlock()
	return
}

func (m *defaultPool) store(key DATABASE_KEY, value *databasePoolItem) {
	m.mu.Lock()
	m.databases[key] = value
	m.mu.Unlock()
}

func (m *defaultPool) Delete(key DATABASE_KEY) {
	m.mu.Lock()
	delete(m.databases, key)
	m.mu.Unlock()
}

func (m *defaultPool) rangeOver(f func(value *databasePoolItem) bool) {
	m.mu.RLock()
	for _, value := range m.databases {
		if !f(value) {
			break
		}
	}
	m.mu.RUnlock()
}

func (m *defaultPool) AutoMigrate() error {
	var err error
	m.rangeOver(func(db *databasePoolItem) bool {
		err = db.db.AutoMigrate(db.models...)
		return err == nil
	})
	return err
}

func (m *defaultPool) Register(key DATABASE_KEY, model ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	db, ok := m.databases[key]
	if !ok {
		return errors.New("database not found")
	}
	db.models = append(db.models, model...)
	return nil
}

func (m *defaultPool) Get(key DATABASE_KEY) (PoolItem[*gorm.DB], error) {
	db, ok := m.load(key)
	if ok {
		return db, nil
	}
	return nil, errors.New("database not found")
}

func (m *defaultPool) Add(key DATABASE_KEY, DB *gorm.DB) error {
	m.store(key, &databasePoolItem{
		db:     DB,
		models: make([]interface{}, 0),
	})
	return nil
}

func (m *defaultPool) ByModel(model interface{}) (PoolItem[*gorm.DB], error) {
	var db PoolItem[*gorm.DB]
	m.rangeOver(func(poolItem *databasePoolItem) bool {
		for _, m := range poolItem.models {
			var aPkg, aName = models.GetMetaData(m)
			var bPkg, bName = models.GetMetaData(model)
			if aPkg == bPkg && aName == bName {
				db = poolItem
				return false
			}
		}
		return true
	})
	if db == nil {
		return nil, errors.New("database not found")
	}
	return db, nil
}

func (m *defaultPool) GetDefaultDB() PoolItem[*gorm.DB] {
	var db PoolItem[*gorm.DB]
	var err error
	db, err = m.Get(DEFAULT_DATABASE_KEY)
	if err != nil || db == nil {
		panic("default database not found, please add default database")
	}
	return db
}
