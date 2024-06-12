import { EditorJSWidget } from './editorjs-widget';

export {};

declare global {
    interface Window {
        EditorJSWidget: typeof EditorJSWidget
        editors: EditorJSWidget[]
    }
}