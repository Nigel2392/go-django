import { MenuController } from "./controllers/menu";
import { DropdownController } from "./controllers/dropdown";
import { TippyController } from "./controllers/tippy";
import { PanelController, TitlePanelController } from "./controllers/panel";

import { AdminSite } from "./app/app";
import { AccordionController } from "./controllers/accordion";
import { TabPanelController } from "./controllers/panel_tab";

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

app.start();

console.log('Admin app started');