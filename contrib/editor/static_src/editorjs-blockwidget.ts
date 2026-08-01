var EditorJSBoundWidget: any;
var EditorJSTelepathWidget: any;

if (window.BoundWidget != undefined) {
    class editorJSBoundWidget extends window.BoundWidget {
        setState(state: any): void {
            // check if json encode required
            if (typeof state === 'object') {
                state = JSON.stringify(state);
            }

            this.input!.value = state;
        }

        setValue(value: any): void {
            // check if json encode required
            if (typeof value === 'object') {
                value = JSON.stringify(value);
            }

            this.input!.value = value;
        }
    }

    class editorJSTelepathWidget extends window.Widget {
        boundWidgetClass = EditorJSBoundWidget;
    }

    EditorJSBoundWidget = editorJSBoundWidget;
    EditorJSTelepathWidget = editorJSTelepathWidget;
}

export { EditorJSTelepathWidget, EditorJSBoundWidget };
