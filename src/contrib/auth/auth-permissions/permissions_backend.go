package auth_permissions

import (
	"errors"
	"net/http"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/attrutils"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/mux/middleware/authentication"
)

var _ (permissions.PermissionTester) = (*PermissionsBackend)(nil)

type PermissionsBackend struct {
	db DBQuerier
}

func NewPermissionsBackend(db DBQuerier) *PermissionsBackend {
	return &PermissionsBackend{db: db}
}

func (pb *PermissionsBackend) HasPermission(r *http.Request, perms ...string) bool {
	return pb.HasObjectPermission(r, nil, perms...)
}

func (pb *PermissionsBackend) HasObjectPermission(r *http.Request, obj interface{}, perms ...string) bool {
	var user = authentication.Retrieve(r)
	if user == nil {
		return false
	}

	if len(perms) == 0 {
		return true
	}

	if !user.IsAuthenticated() {
		return false
	}

	if user.IsAdmin() {
		return true
	}

	var u = user.(attrs.Definer)
	var primary = attrs.PrimaryKey(u)
	if primary == nil {
		return false
	}

	var err error
	primary, err = attrutils.CastToNumber[uint64](primary)
	if err != nil && (errors.Is(err, attrutils.ErrEmptyString) || errors.Is(err, attrutils.ErrConvertingString)) {
		return false
	}

	var (
		ctx = r.Context()
	)
	tx, err := pb.db.Begin(ctx)
	if err != nil {
		return false
	}
	defer tx.Rollback(ctx)

	var (
		querier = pb.db.WithTx(tx)
	)
	defer querier.Close()

	var checkedCount int
	for _, perm := range perms {
		var hasPerm, err = querier.UserHasPermission(
			ctx, primary.(uint64), perm,
		)
		if err != nil {
			return false
		}
		if hasPerm > 0 {
			checkedCount++
		}
	}

	return len(perms) == checkedCount
}
