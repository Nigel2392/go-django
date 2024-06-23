package auditlogs

import (
	"database/sql"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/admin/components/menu"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/goldcrest"
)

type AuditLogs struct {
	*apps.AppConfig
}

var Logs *AuditLogs = &AuditLogs{
	AppConfig: apps.NewAppConfig("auditlogs"),
}

func NewAppConfig() django.AppConfig {
	Logs.Init = func(settings django.Settings) error {

		dbInt, ok := settings.Get("DATABASE")
		assert.True(ok, "DATABASE setting is required for 'auditlogs' app")

		_, ok = dbInt.(*sql.DB)
		assert.True(ok, "DATABASE setting must adhere to auditlogs-models.DBTX interface")

		// Logs.Queries = models.NewQueries(db)
		goldcrest.Register(
			admin.RegisterMenuItemHook, 100,
			admin.RegisterMenuItemHookFunc(func(adminSite *admin.AdminApplication, items menu.Items) {

				var auditLogItem = menu.SubmenuItem{
					BaseItem: menu.BaseItem{
						Label: fields.S("Audit Logs"),
					},
					Menu: &menu.Menu{
						Items: []menu.MenuItem{
							&menu.BaseItem{
								Label: fields.S("View Logs"),
							},
							&menu.SubmenuItem{
								BaseItem: menu.BaseItem{
									Label: fields.S("Audit Logs"),
								},
								Menu: &menu.Menu{
									Items: []menu.MenuItem{
										&menu.BaseItem{
											Label: fields.S("View Logs"),
										},
									},
								},
							},
							&menu.SubmenuItem{
								BaseItem: menu.BaseItem{
									Label: fields.S("Audit Logs"),
								},
								Menu: &menu.Menu{
									Items: []menu.MenuItem{
										&menu.BaseItem{
											Label: fields.S("View Logs"),
										},
										&menu.SubmenuItem{
											BaseItem: menu.BaseItem{
												Label: fields.S("Audit Logs"),
											},
											Menu: &menu.Menu{
												Items: []menu.MenuItem{
													&menu.BaseItem{
														Label: fields.S("View Logs"),
													},
												},
											},
										},
									},
								},
							},
						},
					},
				}

				items.Append(&auditLogItem)
			}),
		)

		return nil
	}

	return Logs
}
