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
import SidebarController from "./controllers/sidebar";
import { InlinePanelController } from "./controllers/panel_inline";

const app = new AdminSite({
    controllers: {
        menu: MenuController,
        tooltip: TippyController,
        dropdown: DropdownController,
        accordion: AccordionController,
        panel: PanelController,
        tabpanel: TabPanelController,
        titlepanel: TitlePanelController,
        "inline-panel": InlinePanelController,
        sidebar: SidebarController,
        "bulk-actions": BulkActionsController,
        "side-panels": SidePanelsController,
    },
});

window.AdminSite = app;

window.getCookie = function(name: string): string | null {
    let cookieValue = null;
    if (document.cookie && document.cookie !== '') {
        const cookies = document.cookie.split(';');
        for (let i = 0; i < cookies.length; i++) {
            const cookie = cookies[i].trim();
            // Does this cookie string begin with the name we want?
            if (cookie.substring(0, name.length + 1) === (name + '=')) {
                cookieValue = decodeURIComponent(cookie.substring(name.length + 1));
                break;
            }
        }
    }
    return cookieValue;
};

window.setCookie = function(name: string, value?: string, days?: number) {
    var expires = "";
    if (days) {
        var date = new Date();
        date.setTime(date.getTime() + (days * 24 * 60 * 60 * 1000));
        expires = "; expires=" + date.toUTCString();
    }
    
    if (!value) {
        value = "";
    }

    document.cookie = name + "=" + (value || "") + expires + "; path=/";
};

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

document.addEventListener("DOMContentLoaded", () => {
    setTimeout(() => {
        document.body.classList.remove("preloading");
    }, 100);
});

console.log('Admin app started');