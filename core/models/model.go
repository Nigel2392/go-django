package models

import (
	"database/sql/driver"
	"encoding/gob"
	"reflect"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func init() {
	gob.Register(&UUIDField{})
}

type AdminAllowedIDConstraint interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | string | UUIDField | uuid.UUID
}

type Model struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement;not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;not null"`
}

type UUIDModel struct {
	ID        UUIDField `gorm:"column:id;primaryKey;type:string;size:36;not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;not null"`
}

func (m *UUIDModel) BeforeCreate(g *gorm.DB) error {
	m.ID = UUIDField(uuid.New())
	return nil
}

type UUIDField uuid.UUID

func (a UUIDField) String() string {
	return uuid.UUID(a).String()
}

func (a *UUIDField) Scan(value interface{}) error {
	var uid uuid.UUID
	var err error
	switch value := value.(type) {
	case string:
		uid, err = uuid.Parse(value)
		if err != nil {
			return err
		}
	case []byte:
		uid, err = uuid.ParseBytes(value)
		if err != nil {
			return err
		}
	default:
		return err
	}
	*a = UUIDField(uid)
	return nil
}

func (a UUIDField) Value() (driver.Value, error) {
	return uuid.UUID(a).String(), nil
}

func (a UUIDField) UUID() uuid.UUID {
	return uuid.UUID(a)
}

func (a UUIDField) Display() string {
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
