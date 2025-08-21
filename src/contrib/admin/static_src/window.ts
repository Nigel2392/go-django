import { AdminSite } from "./app/app";
import { Application } from "@hotwired/stimulus";

export {};

declare global {
    interface Window {
        Stimulus: Application;
        AdminSite: AdminSite;
        i18n: {
            gettext(str: string, ...args: any): string,
            ngettext(singular: string, plural: string, n: any, ...args: any): string,
        };
    }
}