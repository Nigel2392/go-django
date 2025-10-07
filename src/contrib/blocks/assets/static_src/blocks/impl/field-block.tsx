import { BoundBlock, Block, Config } from '../base';
import { jsx } from '../../../../../editor/features/links/static_src/jsx';
import { BoundWidget, Widget } from '../../widgets/widget';

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
               <label for={block.config.id}>{block.config.block.label}</label>
           </div>
        )


        root.appendChild(this.labelWrapper);
        root.appendChild(this.errorList);

        if (block.config.block.helpText) {
           root.appendChild(
               <div class="field-help">{block.config.block.helpText}</div>
           );
        }
        
        const widgetPlaceholder = (
              <div class="field-widget"></div>
        );
        root.appendChild(widgetPlaceholder);

        this.widget = block.config.block.widget.render(
           widgetPlaceholder, block.config.name, block.config.id, initialState, 
        );

        if (block.config.errors && block.config.errors.length) {
           this.setError(block.config.errors);
        }
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
    
    get widget(): Widget {
        return this.config.block.config.widget;
    }

    get label(): string | undefined {
        return this.config.block.config.label;
    }

    get helpText(): string | undefined {
        return this.config.block.config.helpText;
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