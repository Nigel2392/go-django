import { PageLinkTool } from './links';

export {};

declare global {
    interface Window {
        PageLinkTool: typeof PageLinkTool;
    }
}