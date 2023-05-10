package admin

import (
	"fmt"

	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/request/response"
	"github.com/Nigel2392/router/v3/templates/extensions"
	"github.com/gosimple/slug"
)

// Register an extension to the admin site.
//
// # Extensions are separate templates that can be used to add extra functionality
//
// These templates are embedded into the admin site's base template.
func RegisterExtension(ext extensions.Extension) {
	if AdminSite_ExtensionTemplateManager == nil {
		AdminSite_ExtensionTemplateManager = response.TEMPLATE_MANAGER
	}

	if AdminSite_ExtensionOptions == nil {
		AdminSite_ExtensionOptions = &extensions.Options{
			BaseManager:      templateManager(),
			ExtensionManager: AdminSite_ExtensionTemplateManager,
			TemplateName:     "base",
			BlockName:        "content",
		}
	}

	var exts = make([]extensions.Extension, 0)
	for _, adminExtension := range adminSite_Extensions {
		if adminExtension.Name() == ext.Name() {
			AdminSite_Logger.Warningf("admin: extension %s already registered\n", ext.Name())
			return
		}
	}

	adminSite_Extensions = append(adminSite_Extensions, exts...)

	adminSite_ExtensionsRoute.Any(
		fmt.Sprintf("/%s", slug.Make(ext.Name())),
		router.HandleFunc(extensions.View(AdminSite_ExtensionOptions, ext)),
		ext.Name(),
	)
}
