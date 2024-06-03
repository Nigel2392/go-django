import type { Application } from "@hotwired/stimulus";
import { DjangoApplication } from "./app/app";

export {};

declare global {
    interface Window {
        Stimulus: Application;
        Django: DjangoApplication;
    }
}