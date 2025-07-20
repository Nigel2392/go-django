package auth

import (
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"golang.org/x/crypto/bcrypt"
)

var HASHER = func(b string) (string, error) {
	var bytes, err = bcrypt.GenerateFromPassword([]byte(b), bcrypt.DefaultCost)
	return string(bytes), err
}

var CHECKER = func(hashedPassword, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return autherrors.ErrPwdHashMismatch
	}
	return nil
}

func CheckPassword(u *User, password string) error {
	return u.Password.Check(password)
}

var _ driver.Valuer = &Password{}
var _ sql.Scanner = &Password{}

type Password struct {
	Raw  string `json:"raw"`
	raw  string `json:"-"`
	Hash string `json:"hash"`
}

func NewPassword(raw string) *Password {
	return &Password{
		Raw:  raw,
		raw:  raw,
		Hash: "",
	}
}

func NewHashedPassword(hash string) *Password {
	return &Password{
		Raw:  "",
		raw:  "",
		Hash: hash,
	}
}

func (p *Password) DBType() dbtype.Type {
	return dbtype.String
}

func (p *Password) IsZero() bool {
	if p == nil {
		return true
	}
	if p.Hash != "" {
		return false
	}
	if p.Raw != "" {
		return false
	}
	return true
}

func (p *Password) String() string {
	if p.IsZero() {
		return "<nil>"
	}
	if p.Hash != "" {
		return p.Hash
	}
	return "***********"
}

func (p *Password) Check(password string) error {
	if p == nil {
		if password == "" {
			return nil
		}
		return fmt.Errorf(
			"password is nil, but a password was provided: %s: %w",
			password, autherrors.ErrPwdHashMismatch,
		)
	}

	if p.Hash == "" && p.Raw != "" {
		var pw, err = HASHER(p.Raw)
		if err != nil {
			return errors.ValueError.Wrapf(
				"cannot hash password: %v", err,
			)
		}
		p.Hash = pw
		p.raw = p.Raw
	}

	return CHECKER(p.Hash, password)
}

func (p *Password) Scan(value any) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		p.Hash = v
	case Password:
		p.Hash = v.Hash
	case []byte:
		p.Hash = string(v)
	default:
		return errors.ValueError.Wrapf(
			"cannot scan value %T into Password: %v", value, value,
		)
	}

	return nil
}

func (p *Password) Value() (driver.Value, error) {
	if p.Hash != "" {
		return p.Hash, nil
	}

	if p.Raw != "" && (p.Raw != p.raw || p.Hash == "") {
		var pw, err = HASHER(p.Raw)
		if err != nil {
			return nil, errors.ValueError.Wrapf(
				"cannot hash password: %v", err,
			)
		}
		p.Hash = pw
		p.raw = p.Raw
		goto returnHash

	} else if p.Hash == "" && p.Raw == "" {
		return nil, nil
	}

returnHash:
	return p.Hash, nil
}
