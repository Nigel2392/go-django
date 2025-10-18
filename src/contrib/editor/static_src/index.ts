import { EditorJSTelepathWidget } from './editorjs-blockwidget';

export { EditorJSWidget } from './editorjs-widget';
export { EditorJSWidgetController } from './editorjs-widget-controller';

window.telepath.register(
    "editor.widgets.EditorJSWidget",
    EditorJSTelepathWidget,
);