package permissions_models

func FilterGroupPermissions(rows []*UserGroupsRow) (Group, []*Permission) {
	var (
		group       Group
		permissions = make([]*Permission, 0, len(rows))
	)
	for i, row := range rows {
		if i == 0 {
			group = row.Group
		}
		permissions = append(permissions, &row.Permission)
	}
	return group, permissions
}
