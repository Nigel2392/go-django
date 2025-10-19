import { BoundBlock, Block, BlockMeta, copyAttrs } from '../base';
import { BoundWidget, Widget } from '../../widgets/widget';
import { jsx } from '../../../../../admin/static_src/jsx';
import { PanelComponent } from '../../../../../admin/static_src/utils/panels';

class BoundFieldBlock extends BoundBlock<FieldBlock> {
    errorList: HTMLElement;
    labelWrapper: HTMLElement;
    helpText: HTMLElement;
    inputWrapper: HTMLElement;
    widget: BoundWidget;
    attrs: any;

    constructor(block: FieldBlock, placeholder: HTMLElement, name: string, id: string, initialState: any, initialError: any, attrs: any = {}) {

        const errorList = (
           <ul class="field-errors"></ul>
        )

        const labelWrapper = (
           <div class="field-label">
               <label for={id}>{block.meta.label}</label>
           </div>
        )

        const widgetPlaceholder = (
            <div class="field-widget"></div>
        );

        const root = PanelComponent({
            panelId: `${name}--panel`,
            class: "field-block",
            allowPanelLink: !!block.meta.label,
            heading: labelWrapper,
            helpText: block.meta.helpText ? (
                <div class="field-help">{block.meta.helpText}</div>
            ) : null,
            errors: errorList,
            children: widgetPlaceholder,
        });

        copyAttrs(
            placeholder,
            root.hasAttribute.bind(root),
            root.body.setAttribute.bind(root.body),
        );

        placeholder.replaceWith(root);

        super(block, name, root);
        this.errorList = errorList;
        this.labelWrapper = labelWrapper;
        this.attrs = attrs;
        
        const options = {
            attributes: this.getAttributes(),
        };

        initialState = initialState ?? block.meta.default ?? null;

        this.widget = block.widget.render(
           widgetPlaceholder, name, id, initialState, options,
        );

        if (initialError) {
           this.setError(initialError);
        }
    }

    getAttributes(): any {
        const attrs = this.attrs || {};

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
        if (!errors || errors?.length === 0) {
            return;
        }

        if (Array.isArray(errors)) {
            
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
        } else if (typeof errors === 'object') {
            var keys = Object.keys(errors);
            this.errorList.innerHTML = "";
            if (!keys.length) {
                return;
            }
            
            keys.forEach((key: string) => {
                errors[key].forEach((error: string) => {
                    const errorItem = (
                        <li>{error}</li>
                    );
                    this.errorList.appendChild(errorItem);
                });
            })
        } else {
            this.errorList.innerHTML = "";
            this.errorList.appendChild(
                <li>{errors}</li>
            );
        }
    }
}

class FieldBlock extends Block {
    name: string;
    widget: Widget;

    constructor(name: string, widget: Widget, meta: BlockMeta) {
        super();
        this.name = name;
        this.widget = widget;
        this.meta = meta;
    }
    
    render(root: HTMLElement, id: string, name: string, initialState: any, initialError: any, attrs: any = {}): BoundFieldBlock {
        return new BoundFieldBlock(
            this,
            root,
            name,
            id,
            initialState,
            initialError,
            attrs,
        );
    }
}

export {
    BoundFieldBlock,
    FieldBlock,
};