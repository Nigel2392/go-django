package securedb

import (
	"database/sql"
	"database/sql/driver"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
)

// tested in contrib/auth/password_test.go

var _ driver.Valuer = &StringHash[BcryptHasher]{}
var _ sql.Scanner = &StringHash[BcryptHasher]{}

type StringHash[HASHER HasherType] struct {
	Raw  string `json:"raw"`
	raw  string `json:"-"`
	Hash string `json:"hash"`
}

func NewString[HASHER HasherType](raw string) *StringHash[HASHER] {
	return &StringHash[HASHER]{
		Raw:  raw,
		raw:  raw,
		Hash: "",
	}
}

func NewStringHash[HASHER HasherType](hash string) *StringHash[HASHER] {
	return &StringHash[HASHER]{
		Raw:  "",
		raw:  "",
		Hash: hash,
	}
}

func (p *StringHash[H]) DBType() dbtype.Type {
	return dbtype.String
}

func (p *StringHash[H]) IsZero() bool {
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

func (p *StringHash[H]) String() string {
	if p.IsZero() {
		return "<nil>"
	}
	if p.Hash != "" {
		return p.Hash
	}
	return "***********"
}

func (p *StringHash[H]) SetValue(raw string) error {
	var h H
	hash, err := h.HashValue(raw)
	if err != nil {
		return ErrSecureValue.Wrapf("failed to hash value: %v", err)
	}

	p.Raw = raw
	p.raw = raw
	p.Hash = hash
	return nil
}

func (p *StringHash[H]) Check(value string) error {
	if p == nil {
		if value == "" {
			return nil
		}
		return ErrSecureValue.Wrapf(
			"string is nil but a string was provided: %s: %w",
			value, ErrSecureValue,
		)
	}

	var h H
	if p.Hash == "" && p.Raw != "" {
		var pw, err = h.HashValue(p.Raw)
		if err != nil {
			return ErrSecureValue.Wrapf(
				"cannot hash string: %v", err,
			)
		}
		p.Hash = pw
		p.raw = p.Raw
	}

	return h.CheckValueHash(p.Hash, value)
}

func (p *StringHash[H]) Scan(value any) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		p.Hash = v
	case StringHash[H]:
		p.Hash = v.Hash
	case []byte:
		p.Hash = string(v)
	default:
		return ErrSecureValue.Wrapf(
			"cannot scan value %T into StringHash: %v", value, value,
		)
	}

	return nil
}

func (p *StringHash[H]) Value() (driver.Value, error) {
	if p.Hash != "" {
		return p.Hash, nil
	}

	if p.Raw != "" && (p.Raw != p.raw || p.Hash == "") {
		var h H
		var pw, err = h.HashValue(p.Raw)
		if err != nil {
			return nil, ErrSecureValue.Wrapf(
				"cannot hash string: %v", err,
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
