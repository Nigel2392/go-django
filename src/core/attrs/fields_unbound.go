package attrs

import (
	"context"
)

type unboundField struct {
	name string
	cnf  *Config
}

func Unbound(name string, cnf ...*Config) UnboundFieldConstructor {
	if len(cnf) > 0 && cnf[0] != nil {
		return &unboundField{name: name, cnf: cnf[0]}
	}
	return &unboundField{name: name}
}

func (u *unboundField) Name() string {
	if u.cnf != nil && u.cnf.NameOverride != "" {
		return u.cnf.NameOverride
	}
	return u.name
}

func (u *unboundField) BindField(ctx context.Context, d Definer) (Field, error) {
	return NewField(d, u.name, u.cnf), nil
}
