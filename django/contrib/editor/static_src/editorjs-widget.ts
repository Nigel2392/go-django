import EditorJS from "@editorjs/editorjs";

type EditorJSWidgetConfig = {
    holder: string | HTMLElement,
    tools: any,
    data?: any,
    onReady?: () => void,
    onChange?: () => void,
};

type EditorJSWidgetElement = HTMLInputElement & {
    CurrentWidget: EditorJSWidget,
    CurrentEditor: EditorJS,
};

function keepEditorInstance(instance: EditorJSWidget) {
    window.GoEditorJS.editors.push(this);
}

class EditorJSWidget {
    element: EditorJSWidgetElement;
    config: EditorJSWidgetConfig;
    editorConfig: EditorJSWidgetConfig;
    editor: EditorJS;


    constructor(elementWrapper: EditorJSWidgetElement, hiddenInput: EditorJSWidgetElement, config: EditorJSWidgetConfig) {
        this.element = hiddenInput;
        this.config = config;

        hiddenInput.CurrentWidget = this;
        elementWrapper.CurrentWidget = this;

        this.initEditor();
    }

    initEditor() {
        this.editorConfig = {
            ...this.config,
            onReady: async () => {
                const editorData = await this.editor.save();
                this.element.value = JSON.stringify(editorData);

                this.dispatchEvent('ready', {
                    data: editorData,
                });
            },
            onChange: async () => {
                const editorData = await this.editor.save();
                this.element.value = JSON.stringify(editorData);

                this.dispatchEvent('change', {
                    data: editorData,
                });
            },
        };

        if (this.element.value) {
            this.editorConfig.data = JSON.parse(this.element.value);
        }

        console.log('EditorJSWidget initialized with config:', this.editorConfig);

        keepEditorInstance(this)

        var savedForm = false;
        console.log('Adding submit event listener to form');
        var form = this.element.closest('form') as HTMLFormElement;
        console.log(form);
        form.addEventListener('submit', (e: SubmitEvent) => {
            if (savedForm) {
                return;
            }

            e.preventDefault();
            e.stopPropagation();

            this.editor.save().then((outputData) => {
                this.element.value = JSON.stringify(outputData);
                savedForm = true;
            }).catch((reason) => {
                alert(`Failed to save EditorJS data: ${reason}`);
            });
        });

        console.log('Initializing EditorJS with config:', this.editorConfig);

        this.editor = new EditorJS(this.editorConfig);
        this.element.setAttribute('data-editorjs-initialized', 'true');
        this.element.CurrentEditor = this.editor;

        console.log('EditorJS initialized');

        this.editor.isReady.then(() => {

            console.log('EditorJS is ready');

            this.dispatchEvent('ready', {
                data: this.editorConfig.data,
            });

        }).catch((reason) => {

            console.error(`Editor.js failed to initialize: ${reason}`);
            this.dispatchEvent('error', {reason: reason});
            console.log(this.editorConfig)
        
        });
    }

    dispatchEvent(eventName: String, data: any = null) {
        if (!data) {
            data = {};
        };

        data.editor = this.editor;
        data.widget = this;

        const event = new CustomEvent(
            `editorjs:${eventName}`,
            {detail: data},
        );

        this.element.dispatchEvent(event);
    }

    focus() {
        this.editor.focus();
    }

    disconnect() {
        this.editor.destroy();
    }
}

window.GoEditorJS = {
    editors: [],
    Widget: EditorJSWidget,
};

export {
    EditorJSWidget,
    EditorJSWidgetConfig,
    EditorJSWidgetElement,
}