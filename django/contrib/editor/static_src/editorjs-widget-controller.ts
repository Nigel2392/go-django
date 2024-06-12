import { Controller } from "@hotwired/stimulus";
import { EditorJSWidget, EditorJSWidgetConfig, EditorJSWidgetElement } from "./editorjs-widget";

class EditorJSWidgetController extends Controller<HTMLDivElement> {
    declare configValue: EditorJSWidgetConfig;
    declare inputTarget: EditorJSWidgetElement;
    declare widget: EditorJSWidget | null;


    static targets = [
        'input', 'editor',
    ];
    static values = {
        config: Object,
    };

    connect() {
        console.log('EditorJSWidgetController connected', this.configValue);
        const keys = Object.keys(this.configValue.tools);
        for (let i = 0; i < keys.length; i++) {
            const key = keys[i];
            const toolConfig = this.configValue.tools[key];
            const toolClass = window[toolConfig.class];
            toolConfig.class = toolClass;
            this.configValue.tools[key] = toolConfig;
        }

        this.widget = new EditorJSWidget(
            this.element as EditorJSWidgetElement,
            this.inputTarget,
            this.configValue,
        );
    }

    disconnect() {
        this.widget?.disconnect();
        this.widget = null;
    }
}

window.Stimulus.register('editorjs-widget', EditorJSWidgetController);

export { EditorJSWidgetController };