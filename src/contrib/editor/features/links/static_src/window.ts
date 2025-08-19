import { Chooser } from '../../../../admin/chooser/static_src/chooser/chooser';
import { PageLinkTool } from './links';

export {};

declare global {
    interface Window {
        PageLinkTool: typeof PageLinkTool;
        Chooser:      typeof Chooser;
    }
}