import { MenuController } from "./controllers/menu";
import { DropdownController } from "./controllers/dropdown";
import { TippyController } from "./controllers/tippy";
import { PanelController, TitlePanelController } from "./controllers/panel";

import { AdminSite } from "./app/app";
import { AccordionController } from "./controllers/accordion";
import { TabPanelController } from "./controllers/panel_tab";
import sprintf from "./utils/sprintf";

const app = new AdminSite({
    controllers: {
        menu: MenuController,
        tooltip: TippyController,
        dropdown: DropdownController,
        accordion: AccordionController,
        panel: PanelController,
        tabpanel: TabPanelController,
        titlepanel: TitlePanelController,
    },
});

window.AdminSite = app;

if (!window.i18n || (!window.i18n.gettext && !window.i18n.ngettext)) {
    window.i18n = {
        gettext: (str: string, ...args: any) => sprintf(str, ...args),
        ngettext: (singular: string, plural: string, n: any, ...args: any) => sprintf(n === 1 ? singular : plural, ...args),
    };
}

window.sprintf = sprintf;

app.start();

console.log('Admin app started');