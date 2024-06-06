import { Block, BlockDef, Config } from '../base';
import { jsx } from '../../components/jsx';

function toElement(html: string): HTMLElement {
    const template = document.createElement('template');
    template.innerHTML = html.trim();
    return template.content.firstChild as HTMLElement;
}

class FieldBlock extends Block {
    errorList: HTMLElement;
    labelWrapper: HTMLElement;
    inputWrapper: HTMLElement;
    input: HTMLInputElement;

    constructor(config: Config, widget: HTMLElement, blockDef: BlockDef) {
        super(widget, blockDef);

        console.log("FieldBlock constructor", config);

        this.errorList = (
            <ul class="field-errors"></ul>
        )

        this.labelWrapper = (
            <div class="field-label">
                <label for={config.id}>{config.block.element.label}</label>
            </div>
        )

        const inputHtml = toElement(config.block.element.html.replace(
            "__PREFIX__", config.name,
        ).replace(
            "__ID__", config.id,
        ))

        this.input = inputHtml.querySelector('input');
        this.widget.appendChild(this.errorList);
        this.widget.appendChild(this.labelWrapper);
        this.widget.appendChild(
            <div class="field-input">{ inputHtml }</div>
        );

        if (config.block.element.helpText) {
            this.widget.appendChild(
                <div class="field-help">{config.block.element.helpText}</div>
            );
        }

        if (config.errors && config.errors.length) {
            this.setError(config.errors);
        }
    }

    getLabel(): string {
        return this.input.value;
    }

    getState(): any {
        return this.input.value;
    }

    setState(state: any): void {
        this.input.value = state;
    }

    setError(errors: any): void {
        if (!errors) {
            return;
        }
        if (errors instanceof Array) {
            
            if (!errors.length) {
                this.errorList.innerHTML = "";
                return;
            }
            
            this.errorList.innerHTML = "";
            errors.forEach((error: string) => {
                const errorItem = (
                    <li>{error}</li>
                );
                this.errorList.appendChild(errorItem);
            });
        } else {
            this.errorList.innerHTML = "";
            this.errorList.appendChild(
                <li>{errors}</li>
            );
        }
    }
}

class FieldBlockDef extends BlockDef {
    constructor(element: HTMLElement, config: Config) {
        super(element, config);
        console.log("FieldBlockDef constructor", element, config);
    }

    render(): any {
        return new FieldBlock(
            this.config,
            this.element,
            this,
        );
    }

    config: Config;
    element: HTMLElement;
}

export {
    FieldBlock,
    FieldBlockDef,
};