
class EditorJSBoundWidget extends window.BoundWidget {
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

class EditorJSTelepathWidget extends window.Widget {
    boundWidgetClass = EditorJSBoundWidget;
}

export { EditorJSTelepathWidget, EditorJSBoundWidget };
