import { EditorJSTelepathWidget } from './editorjs-blockwidget';

export { EditorJSWidget } from './editorjs-widget';
export { EditorJSWidgetController } from './editorjs-widget-controller';

if (EditorJSTelepathWidget) {
    window.telepath.register(
        "editor.widgets.EditorJSWidget",
        EditorJSTelepathWidget,
    );
}
