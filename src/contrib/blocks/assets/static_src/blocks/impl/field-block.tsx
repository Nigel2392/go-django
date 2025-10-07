import { BoundBlock, Block, Config } from '../base';
import { BoundWidget, Widget } from '../../widgets/widget';
import { jsx } from '../../../../../editor/features/links/static_src/jsx';

class BoundFieldBlock extends BoundBlock<FieldBlock> {
    errorList: HTMLElement;
    labelWrapper: HTMLElement;
    helpText: HTMLElement;
    inputWrapper: HTMLElement;
    widget: BoundWidget;

    constructor(block: FieldBlock, root: HTMLElement, name: String, initialState: any, initialError: any) {
        super(block, name);

        this.errorList = (
           <ul class="field-errors"></ul>
        )

        this.labelWrapper = (
           <div class="field-label">
               <label for={block.id}>{block.label}</label>
           </div>
        )


        root.appendChild(this.labelWrapper);
        root.appendChild(this.errorList);

        if (block.helpText) {
           root.appendChild(
               <div class="field-help">{block.helpText}</div>
           );
        }
        
        const widgetPlaceholder = (
              <div class="field-widget"></div>
        );
        root.appendChild(widgetPlaceholder);

        const options = {
            attributes: this.getAttributes(),
        };

        this.widget = block.widget.render(
           widgetPlaceholder, block.name, block.id, initialState, options,
        );

        if (block.errors && block.errors.length) {
           this.setError(block.errors);
        }
    }

    getAttributes(): any {
        const attrs = this.block.attrs || {};

        if (this.block.meta.required) {
            attrs['required'] = 'required';
        }

        return attrs;
    }

    getLabel(): string {
        return this.widget.getTextLabel();
    }

    getState(): any {
        return this.widget.getState();
    }

    setState(state: any): void {
        this.widget.setState(state);
    }

    getValue(): any {
        return this.widget.getValue();
    }

    setValue(value: any): void {
        this.widget.setValue(value);
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

class FieldBlock extends Block<FieldBlock> {
    constructor(config: Config) {
        super(config);
    }
    
    get id(): string {
        return this.config.id;
    }

    get name(): string {
        return this.config.name;
    }

    get errors(): any {
        return this.config.errors;
    }

    get attrs(): { [key: string]: any } | undefined {
        return this.config.attrs;
    }

    get widget(): Widget {
        return this.meta.widget;
    }

    get label(): string | undefined {
        return this.meta.label;
    }

    get helpText(): string | undefined {
        return this.meta.helpText;
    }

    get meta(): any {
        return this.config.block.config;
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
}

export {
    BoundFieldBlock,
    FieldBlock,
};