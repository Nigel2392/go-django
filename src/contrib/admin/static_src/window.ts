import { AdminSite } from "./app/app";
import { Application } from "@hotwired/stimulus";
import sprintf from "./utils/sprintf";

export {};

declare global {
    interface Window {
        Stimulus: Application;
        AdminSite: AdminSite;
        sprintf: typeof sprintf;
        i18n: {
            gettext(str: string, ...args: any): string,
            ngettext(singular: string, plural: string, n: any, ...args: any): string,
        };
    }
}