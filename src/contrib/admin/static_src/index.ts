import { MenuController } from "./controllers/menu";
import { DropdownController } from "./controllers/dropdown";
import { TippyController } from "./controllers/tippy";
import { PanelController } from "./controllers/panel";

import { AdminSite } from "./app/app";
import { AccordionController } from "./controllers/accordion";

const app = new AdminSite({
    controllers: {
        menu: MenuController,
        tooltip: TippyController,
        dropdown: DropdownController,
        accordion: AccordionController,
        panel: PanelController,
    },
});

window.AdminSite = app;

app.start();

console.log('Admin app started');