package auth

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Nigel2392/go-datastructures/linkedlist"
	"github.com/Nigel2392/go-django/core/views/fields"
	"github.com/Nigel2392/go-django/core/views/interfaces"
)

type Group struct {
	ID          int64            `admin-form:"readonly;disabled;omit_on_create;" gorm:"-" json:"id"`
	Name        string           `gorm:"-" json:"name"`
	Description fields.TextField `admin-form:"textarea;" gorm:"-" json:"description"`

	// Non-SQLC fields
	Permissions      []Permission                     `admin-form:"-" gorm:"-" json:"permissions"`
	PermissionSelect fields.DoubleMultipleSelectField `admin-form:"-" gorm:"-" json:"permission_select"`
	Users            []User                           `admin-form:"-" gorm:"-" json:"users"`
}

func (g *Group) String() string {
	return g.Name
}

func (g *Group) Save(creating bool) error {
	var ranQueryPermissions, ranQueryGroup bool
	var err error
	var permissions = g.PermissionSelect.Left
	var permissionsIDs = make([]int64, len(permissions))
	for i, group := range permissions {
		var intID, err = strconv.ParseInt(group.Value(), 10, 64)
		if err != nil {
			return err
		}
		permissionsIDs[i] = intID
	}
tryAgain:
	if g.ID == 0 {
		goto createFirst
	} else {
		var err = Auth.Queries.OverrideGroupPermissions(context.Background(), g.ID, permissionsIDs)
		if err != nil {
			return err
		}
		ranQueryPermissions = true
	}
createFirst:
	if !ranQueryGroup {
		if creating {
			err = Auth.Queries.CreateGroup(context.Background(), g)
		} else {
			err = Auth.Queries.UpdateGroup(context.Background(), g)
		}
		ranQueryGroup = true
	}
	if !ranQueryPermissions {
		goto tryAgain
	}
	return err
}

func (g *Group) Delete() error {
	return Auth.Queries.DeleteGroup(context.Background(), g.ID)
}

func (p *Group) StringID() string {
	return fmt.Sprintf("%d", p.ID)
}

func (p *Group) GetFromStringID(id string) (*Group, error) {
	var intID, err = strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, err
	}
	return Auth.Queries.GetGroupByID(context.Background(), intID)
}

func (u *Group) List(page, each_page int) ([]*Group, int64, error) {
	var count int64
	var err error
	var groups *linkedlist.Doubly[Group]

	groups, err = Auth.Queries.GetGroupsWithPagination(context.Background(), PaginationParams{
		Offset: int32((page - 1) * each_page),
		Limit:  int32(each_page),
	})
	if err != nil {
		return nil, 0, err
	}

	groupsSlice := make([]*Group, 0, groups.Len())
	for groups.Len() > 0 {
		var p = groups.Shift()
		groupsSlice = append(groupsSlice, &p)
	}

	count, err = Auth.Queries.CountGroups(context.Background())
	if err != nil {
		return nil, 0, err
	}

	return groupsSlice, count, nil
}

func (g *Group) GetGroupSelectLabel() string {
	return "Groups"
}

func (g *Group) GetPermissionSelectOptions() (thisOptions, otherOptions []interfaces.Option) {
	var thisGroups, err = Auth.Queries.GetPermissionsByGroupID(context.Background(), g.ID)
	if err != nil {
		return nil, nil
	}
	var allGroups, err2 = Auth.Queries.PermissionsNotInGroup(context.Background(), g.ID)
	if err2 != nil {
		return nil, nil
	}
	thisOptions = make([]interfaces.Option, 0, thisGroups.Len())
	otherOptions = make([]interfaces.Option, 0, allGroups.Len())
	for n := thisGroups.Head(); n != nil; n = n.Next() {
		var g = n.Value()
		var ptrG = &g
		thisOptions = append(thisOptions, fields.Option{
			Val:  ptrG.StringID(),
			Text: g.Name,
		})
	}
	for n := allGroups.Head(); n != nil; n = n.Next() {
		var g = n.Value()
		var ptrG = &g
		otherOptions = append(otherOptions, fields.Option{
			Val:  ptrG.StringID(),
			Text: g.Name,
		})
	}

	return thisOptions, otherOptions
}
