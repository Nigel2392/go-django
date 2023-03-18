package models

import (
	"database/sql/driver"
	"reflect"
	"time"

	"github.com/google/uuid"
)

type Model[T AdminAllowedModelInterface] struct {
	ID        T         `gorm:"primarykey column:id"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;not null"`
}

type AdminAllowedModelInterface interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | string | DefaultIDField | uuid.UUID
}

type DefaultIDField uuid.UUID

func (a DefaultIDField) String() string {
	return uuid.UUID(a).String()
}

func (a *DefaultIDField) Scan(value interface{}) error {
	var uid = uuid.UUID(*a)
	var err = (&(uid)).Scan(value)
	if err != nil {
		return err
	}
	*a = DefaultIDField(uid)
	// 	var uid uuid.UUID
	// 	var err error
	// 	switch value := value.(type) {
	// 	case string:
	// uid, err = uuid.Parse(value)
	// if err != nil {
	// return err
	// }
	// 	case []byte:
	// uid, err = uuid.ParseBytes(value)
	// if err != nil {
	// return err
	// }
	// 	default:
	// return err
	// 	}
	// 	*a = DefaultIDField(uid)
	// 	return nil

	return nil
}

func (a DefaultIDField) Value() (driver.Value, error) {
	return uuid.UUID(a).Value()
}

func (a DefaultIDField) UUID() uuid.UUID {
	return uuid.UUID(a)
}

func (a DefaultIDField) Display() string {
	return uuid.UUID(a).String()
}

// Get the model's name and package path.
func GetMetaData(model any) (name, pkgpath string) {
	var typ = reflect.TypeOf(model)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ.Name(), typ.PkgPath()
}
