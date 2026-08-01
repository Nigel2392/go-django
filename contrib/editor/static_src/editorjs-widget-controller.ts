import { Controller } from "@hotwired/stimulus";
import { EditorJSWidget, EditorJSWidgetConfig, EditorJSWidgetElement } from "./editorjs-widget";

class EditorJSWidgetController extends Controller<HTMLDivElement> {
    declare configValue: EditorJSWidgetConfig;
    declare inputTarget: EditorJSWidgetElement;
    declare editorTarget: EditorJSWidgetElement;
    declare widget: EditorJSWidget | null;


    static targets = [
        'input', 'editor',
    ];
    static values = {
        config: Object,
    };

    connect() {

        if (this.element.classList.contains('editorjs-widget--connected')) {
            console.warn("EditorJSWidgetController already connected");
            return;
        }

        this.element.classList.add('editorjs-widget--connected');

        let cfg = this.configValue;
        const keys = Object.keys(cfg.tools);
        for (let i = 0; i < keys.length; i++) {
            const key = keys[i];
            const toolConfig = cfg.tools[key];
            if (!toolConfig.class || !(toolConfig.class in window)) {
                console.error(`Tool class not found in window`, toolConfig);
                continue;
            }
            const toolClass = window[toolConfig.class];
            toolConfig.class = toolClass;
            cfg.tools[key] = toolConfig;
        }

        cfg.minHeight = cfg.minHeight || 150;

        //  // add the editor target to the config
        //  cfg["holder"] = this.editorTarget;

        this.widget = new EditorJSWidget(
            this.element as EditorJSWidgetElement,
            this.inputTarget,
            cfg,
        );
    }

    disconnect() {
        // this.widget?.disconnect();
        // this.widget = null;
    }
}

document.addEventListener('DOMContentLoaded', () => {

    window.Stimulus.register('editorjs-widget', EditorJSWidgetController);

});

export { EditorJSWidgetController };