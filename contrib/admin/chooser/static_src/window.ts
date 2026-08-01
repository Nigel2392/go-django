import { Chooser } from "./chooser/chooser";

export {};

declare global {
    interface Window {
        Chooser: typeof Chooser;
    }
}