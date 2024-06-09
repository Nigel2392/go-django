import { AdminSite } from "./app/app";
import { Application } from "@hotwired/stimulus";

export {};

declare global {
    interface Window {
        Stimulus: Application;
        AdminSite: AdminSite;
    }
}