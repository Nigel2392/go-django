import { Block, BlockDef, Config } from '../base';
import { jsx } from '../../../../../editor/features/links/static_src/jsx';

function toElement(html: string): HTMLElement {
    const template = document.createElement('template');
    template.innerHTML = html.trim();
    return template.content.firstChild as HTMLElement;
}

class FieldBlock extends Block {
    errorList: HTMLElement;
    labelWrapper: HTMLElement;
    helpText: HTMLElement;
    inputWrapper: HTMLElement;
    input: HTMLInputElement;

    constructor(blockDef: BlockDef, placeHolder: HTMLElement, prefix: String, initialState: any, initialError: any) {
        super(blockDef, prefix);

        console.log("FieldBlock constructor", blockDef, prefix);

        this.errorList = (
           <ul class="field-errors"></ul>
        )

        this.labelWrapper = (
           <div class="field-label">
               <label for={blockDef.config.id}>{blockDef.config.block.element.label}</label>
           </div>
        )

        this.helpText = (
              <div class="field-help">{blockDef.config.block.element.helpText}</div>
          )

        const inputHtml = toElement(blockDef.config.block.element.html.replace(
           "__PREFIX__", blockDef.config.name,
        ).replace(
           "__ID__", blockDef.config.id,
        ))

        this.input = inputHtml.querySelector('input');
        placeHolder.appendChild(this.errorList);
        placeHolder.appendChild(this.labelWrapper);
        placeHolder.appendChild(this.helpText);
        placeHolder.appendChild(
           <div class="field-input">{ inputHtml }</div>
        );

        if (blockDef.config.block.element.helpText) {
           placeHolder.appendChild(
               <div class="field-help">{blockDef.config.block.element.helpText}</div>
           );
        }
        
        if (blockDef.config.errors && blockDef.config.errors.length) {
           this.setError(blockDef.config.errors);
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
        // console.log("FieldBlockDef constructor", element, config);
    }

    render(placeholder: HTMLElement, prefix: String, initialState: any, initialError: any): any {
        return new FieldBlock(
            this,
            placeholder,
            prefix,
            initialState,
            initialError,
        );
    }

    config: Config;
    element: HTMLElement;
}

export {
    FieldBlock,
    FieldBlockDef,
};