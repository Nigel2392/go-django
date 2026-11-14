package users

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type Permission struct {
	models.Model     `table:"auth_permissions" json:"-"`
	ID               uint64                                    `json:"id"`
	Name             string                                    `json:"name"`
	Description      string                                    `json:"description"`
	GroupPermissions *queries.RelM2M[*Group, *GroupPermission] `json:"-"`
	UserPermissions  *queries.RelM2M[User, *UserPermission]    `json:"-"`
}

func CreatePermissions(ctx context.Context, permissions ...*Permission) (createdAny bool, err error) {
	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		return false, errors.Wrap(err, "CreatePermissions/StartTransaction")
	}

	defer tx.Rollback(ctx)

	permissionNames := make([]string, 0, len(permissions))
	for _, perm := range permissions {
		permissionNames = append(permissionNames, perm.Name)
	}

	qs := queries.GetQuerySetWithContext(ctx, &Permission{})
	n, r, err := qs.
		Select("Name").
		Filter("Name__in", permissionNames).
		IterAll()
	if err != nil {
		return false, errors.Wrap(err, "CreatePermissions/IterAll")
	}

	if n == len(permissions) {
		return false, nil
	}

	exists := make(map[string]struct{}, n)
	for l, err := range r {
		if err != nil {
			return false, errors.Wrap(err, "CreatePermissions/iterExisting")
		}

		exists[l.Object.Name] = struct{}{}
	}

	if len(exists) == len(permissions) {
		return false, nil
	}

	newPerms := make([]*Permission, 0, len(permissions)-len(exists))
	for _, perm := range permissions {
		_, ok := exists[perm.Name]
		if ok {
			continue
		}

		newPerms = append(newPerms, perm)
	}

	_, err = qs.BulkCreate(newPerms)
	if err != nil {
		return false, errors.Wrap(err, "CreatePermissions/BulkCreate")
	}

	err = tx.Commit(ctx)
	if err != nil {
		return false, errors.Wrap(err, "CreatePermissions/Commit")
	}

	return true, nil
}

func (p *Permission) String() string {
	return p.Name
}

func (p *Permission) FieldDefs(ctx context.Context) attrs.Definitions {
	var fields = []attrs.Field{
		attrs.NewField(p, "ID", &attrs.FieldConfig{
			ReadOnly: true,
			Primary:  true,
		}),
		attrs.NewField(p, "Name", &attrs.FieldConfig{
			Label:     "Permission Name",
			HelpText:  trans.S("Name of the permission. This is the name that will be displayed in the UI."),
			MaxLength: 255,
		}),
		attrs.NewField(p, "Description", &attrs.FieldConfig{
			Blank:     true,
			Label:     "Description",
			HelpText:  trans.S("Description of the permission. This is the description that will be displayed in the UI."),
			MaxLength: 1024,
		}),
		fields.NewManyToManyField[*queries.RelM2M[*Group, *GroupPermission]](
			p, "GroupPermissions", &fields.FieldConfig{
				DataModelFieldConfig: fields.DataModelFieldConfig{
					Label:    trans.S("Groups"),
					HelpText: trans.S("The groups this permission is assigned to."),
				},
				ScanTo:            &p.GroupPermissions,
				IsReverse:         true,
				NoReverseRelation: true,
				Rel: attrs.Relate(
					&Group{}, "",
					&attrs.ThroughModel{
						This:   &GroupPermission{},
						Source: "PermissionID",
						Target: "GroupID",
					},
				),
			},
		),
		fields.NewManyToManyField[*queries.RelM2M[User, *UserPermission]](
			p, "UserPermissions", &fields.FieldConfig{
				DataModelFieldConfig: fields.DataModelFieldConfig{
					Label:    trans.S("Users"),
					HelpText: trans.S("The users this permission is assigned to."),
				},
				ScanTo:            &p.UserPermissions,
				IsReverse:         true,
				NoReverseRelation: true,
				Rel: attrs.RelatedDeferred(
					attrs.RelManyToOne,
					MODEL_KEY,
					"", &attrs.ThroughModel{
						This:   &UserPermission{},
						Source: "PermissionID",
						Target: "UserID",
					},
				),
			},
		),
	}
	return p.Model.Define(ctx, p, fields)
}
