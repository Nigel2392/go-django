import { MenuController } from "./controllers/menu";
import { AdminSite } from "./app/app";

const app = new AdminSite({
    controllers: {
        menu: MenuController,
    },
});

window.AdminSite = app;

app.start();

console.log('Admin app started');