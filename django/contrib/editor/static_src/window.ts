import { EditorJSWidget } from './editorjs-widget';

export {};

declare global {
    interface Window {
        GoEditorJS: {
            Widget: typeof EditorJSWidget
            editors: EditorJSWidget[]
        }
    }
}