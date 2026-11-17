package auth

import (
	"database/sql"
	"database/sql/driver"

	autherrors "github.com/Nigel2392/go-django/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/queries/src/drivers/securedb"
	"golang.org/x/crypto/bcrypt"
)

type BcryptHasher struct{}

func (h BcryptHasher) HashValue(b string) (string, error) {
	var bytes, err = bcrypt.GenerateFromPassword([]byte(b), bcrypt.DefaultCost)
	return string(bytes), err
}

func (h BcryptHasher) CheckValueHash(hashedPassword, value string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(value)); err != nil {
		return autherrors.ErrPwdHashMismatch
	}
	return nil
}

var _ driver.Valuer = &Password{}
var _ sql.Scanner = &Password{}

type Password = securedb.StringHash[BcryptHasher]

func NewPassword(raw string) *Password {
	return securedb.NewString[BcryptHasher](raw)
}

func NewHashedPassword(hash string) *Password {
	return securedb.NewStringHash[BcryptHasher](hash)
}

func CheckPassword(u *User, password string) error {
	return u.Password.Check(password)
}
