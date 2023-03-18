package db

import (
	"github.com/Nigel2392/go-django/core/httputils/safemap"
	"gorm.io/gorm"
)

var db_map safemap.Map[DATABASE_KEY, PoolItem[*gorm.DB]]

func HasDefaultDB(db_key DATABASE_KEY, p Pool[*gorm.DB]) bool {
	if p == nil {
		return false
	}
	var db, err = p.Get(db_key)
	if err != nil {
		return false
	}
	return db != nil
}

func GetDefaultDatabase(db_key DATABASE_KEY, pool Pool[*gorm.DB]) PoolItem[*gorm.DB] {
	var db PoolItem[*gorm.DB]
	var ok bool

	if db, ok = db_map.Load(db_key); ok {
		return db
	}
	if !HasDefaultDB(db_key, pool) {
		db = pool.GetDefaultDB()
	} else {
		db, _ = pool.Get(db_key)
	}
	db_map.Store(db_key, db)
	return db
}
