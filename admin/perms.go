package admin

import (
	"github.com/Nigel2392/go-django/auth"
)

var (
	PermissionViewAdminSite = &auth.Permission{
		Name:        "view_admin_site",
		Description: "Permission to view the admin site",
	}
	PermissionViewAdminInternal = &auth.Permission{
		Name:        "view_admin_internal",
		Description: "Permission to view the admin internal site",
	}
	PermissionViewAdminExtensions = &auth.Permission{
		Name:        "view_admin_extensions",
		Description: "Permission to view the admin extensions site",
	}
)
