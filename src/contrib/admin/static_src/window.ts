import { AdminSite } from "./app/app";
import { Application, Controller } from "@hotwired/stimulus";
import sprintf from "./utils/sprintf";

export {};

declare global {
    interface Window {
        Stimulus: Application;
        StimulusController: typeof Controller;
        AdminSite: AdminSite;
        //  telepath: {
        //      register(name: any, constructor: any): void;
        //      unpack(objData: any): any;
        //  };
        getCookie: (name: string) => string | null;
        setCookie: (name: string, value: string, days: number) => void;
        sprintf: typeof sprintf;
        i18n: {
            gettext(str: string, ...args: any): string,
            ngettext(singular: string, plural: string, n: any, ...args: any): string,
        };
    }
}