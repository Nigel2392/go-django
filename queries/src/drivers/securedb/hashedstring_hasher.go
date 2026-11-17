package securedb

import "golang.org/x/crypto/bcrypt"

type HasherType interface {
	HashValue(b string) (string, error)
	CheckValueHash(hashedPassword, value string) error
}

type BcryptHasher struct{}

func (h BcryptHasher) HashValue(b string) (string, error) {
	var bytes, err = bcrypt.GenerateFromPassword([]byte(b), bcrypt.DefaultCost)
	return string(bytes), err
}

func (h BcryptHasher) CheckValueHash(hashedPassword, value string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(value)); err != nil {
		return ErrSecureValue
	}
	return nil
}
