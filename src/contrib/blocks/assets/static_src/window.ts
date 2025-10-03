import { BlockApp } from "./app";

export {};

declare global {
    interface Window {
        blocks: BlockApp;
    }
}