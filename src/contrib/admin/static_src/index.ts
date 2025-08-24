import { MenuController } from "./controllers/menu";
import { DropdownController } from "./controllers/dropdown";
import { TippyController } from "./controllers/tippy";
import { PanelController, TitlePanelController } from "./controllers/panel";

import { AdminSite } from "./app/app";
import { AccordionController } from "./controllers/accordion";
import { TabPanelController } from "./controllers/panel_tab";
import sprintf from "./utils/sprintf";
import BulkActionsController from "./controllers/bulk_actions";
import SidePanelsController from "./controllers/side_panels";

const app = new AdminSite({
    controllers: {
        menu: MenuController,
        tooltip: TippyController,
        dropdown: DropdownController,
        accordion: AccordionController,
        panel: PanelController,
        tabpanel: TabPanelController,
        titlepanel: TitlePanelController,
        "bulk-actions": BulkActionsController,
        "side-panels": SidePanelsController,
    },
});

window.AdminSite = app;

if (!window.i18n || (!window.i18n.gettext && !window.i18n.ngettext)) {
    window.i18n = {
        gettext: (str: string, ...args: any) => sprintf(str, ...args),
        ngettext: (singular: string, plural: string, n: any, ...args: any) => {
            var nTyp = typeof n;
            switch (nTyp) {
                case "number":
                    return window.sprintf(n === 1 ? singular : plural, ...args);
                case "object":
                    if (Array.isArray(n)) {
                        return window.sprintf(n.length === 1 ? singular : plural, ...args);
                    }
                    break;
            }
            return window.sprintf(n === 1 ? singular : plural, ...args);
        },
    };
}

window.sprintf = sprintf;

app.start();

console.log('Admin app started');