import { MenuController } from "./controllers/menu";
import { DropdownController } from "./controllers/dropdown";
import { TippyController } from "./controllers/tippy";

import { AdminSite } from "./app/app";

const app = new AdminSite({
    controllers: {
        menu: MenuController,
        tooltip: TippyController,
        dropdown: DropdownController,
    },
});

window.AdminSite = app;

app.start();

console.log('Admin app started');