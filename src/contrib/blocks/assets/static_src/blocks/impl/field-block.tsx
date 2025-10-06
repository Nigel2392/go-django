import { BoundBlock, Block, Config } from '../base';
import { jsx } from '../../../../../editor/features/links/static_src/jsx';

function toElement(html: string): HTMLElement {
    const template = document.createElement('template');
    template.innerHTML = html.trim();
    return template.content.firstChild as HTMLElement;
}

class BoundFieldBlock extends BoundBlock {
    errorList: HTMLElement;
    labelWrapper: HTMLElement;
    helpText: HTMLElement;
    inputWrapper: HTMLElement;
    input: HTMLInputElement;

    constructor(block: Block, placeHolder: HTMLElement, name: String, initialState: any, initialError: any) {
        console.log("FieldBlock constructor", block, name, initialState, initialError);
        super(block, name);

        this.errorList = (
           <ul class="field-errors"></ul>
        )

        this.labelWrapper = (
           <div class="field-label">
               <label for={block.config.id}>{block.config.block.element.label}</label>
           </div>
        )

        const inputHtml = toElement(block.config.block.element.html.replace(
           "__PREFIX__", block.config.name,
        ).replace(
           "__ID__", block.config.id,
        ))

        this.input = inputHtml.querySelector('input');
        placeHolder.appendChild(this.labelWrapper);
        placeHolder.appendChild(this.errorList);

        if (block.config.block.element.helpText) {
           placeHolder.appendChild(
               <div class="field-help">{block.config.block.element.helpText}</div>
           );
        }
        
        placeHolder.appendChild(
           <div class="field-input">{ inputHtml }</div>
        );

        if (block.config.errors && block.config.errors.length) {
           this.setError(block.config.errors);
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

class FieldBlock extends Block {
    constructor(element: HTMLElement, config: Config) {
        super(element, config);
        // console.log("FieldBlockDef constructor", element, config);
    }

    render(root: HTMLElement, name: String, initialState: any, initialError: any): any {
        return new BoundFieldBlock(
            this,
            root,
            name,
            initialState,
            initialError,
        );
    }

    config: Config;
    element: HTMLElement;
}

export {
    BoundFieldBlock,
    FieldBlock,
};