
import { jsx } from '../../../admin/static_src/jsx';

class BoundSupplierBlock extends window.BoundSequenceBlock<SupplierBlock> {
    disallowAdd: Boolean = true;

    constructor(block: any, placeholder: HTMLElement, name: String, id: String, initialState: any, initialError: any) {
        const supplierInput = (document.getElementById("id_Supplier") as HTMLInputElement)
        if (initialState.length == 0 || initialState[0].type != supplierInput.value) {
            const newValue: any = {
                id: '',
                type: supplierInput.value,
                data: null,
            };

            if (block.defaults[supplierInput.value]) {
                newValue.data = block.defaults[supplierInput.value];
            }

            initialState = [newValue]
        }

        super(block, placeholder, name, id, initialState, initialError, {
            allowAdd: false,
            allowSort: false,
            controllable: false,
            hideItemHeading: true,
        });

        console.log(this.getState())

        supplierInput.addEventListener("change", this._changeSupplier.bind(this))
    }

    _changeSupplier(ev: Event) {
        const supplierType = (ev.target as HTMLInputElement).value;
        const childBlock = this.block.childBlocks.get(supplierType);

        // Delete existing block HTML and reset sequence state
        this.itemWrapper.replaceChildren();
        this.items = [];
        this.activeItems = 0;
        this.totalInput.value = '0';

        if (!childBlock) {
            console.warn("Couldn't find child block for supplier:", supplierType);
            return;
        }

        const newValue: any = {
            id: '',
            type: supplierType,
            data: this.block.defaults?.[supplierType] ?? null,
        };

        this._createChild(
            0, 0, this.id, this.name, newValue, null, true, childBlock,
        );

        this.activeItems = 1;
        this.totalInput.value = '1';
    }

    _getChildPanelContentPlaceholder(controls: HTMLElement, actions: HTMLElement): HTMLElement {
        return (
            <div data-sequence-block-field-content class="suppliers-block-field-content">
                <div data-sequence-block-field-placeholder></div>
            </div>
        )
    }

    _getChildBlock(value: any): any | undefined {
        return this.block.childBlocks.get(value.type);
    }

    _renderBoundBlock(child: any, placeholder: HTMLElement, id: string, name: String, value: any, error: any): any {
        return new window.BoundStreamBlockValue(value.id, value.type, child.render(
            placeholder,
            id,
            name,
            value.data,
            error,
        ));
    }

    _defaultChildState(block: any & { name: string }): any {
        const newValue: any = {
            id: '',
            type: block.name,
            data: null,
        };
        if (this.block.defaults[block.name]) {
            newValue.data = this.block.defaults[block.name];
        }
        return newValue;
    }

    _getChildInputs(id: String, name: String, suffix: number, value: any): any {
        const typeKey = `${name}-${suffix}--type`;
        return (
            <input type="hidden" id={ typeKey } name={ typeKey } value={ value.type } />
        );
    }
}

type childBlockMap = {
    name: string;
    block: any;
}[];

class SupplierBlock extends window.Block {
    name: string;
    defaults: { [key: string]: any };
    childBlocks: Map<string, any>;

    constructor(name: string, childBlocks: childBlockMap, defaults: { [key: string]: any }, meta: any) {
        super();
        this.name = name;
        this.meta = meta;
        this.childBlocks = new Map<string, any>();
        this.defaults = defaults;
        for (let i = 0; i < childBlocks.length; i++) {
            this.childBlocks.set(childBlocks[i].name, childBlocks[i].block);
        }
    }

    render(root: HTMLElement, id: String, name: String, initialState: any, initialError: any): any {
        return new BoundSupplierBlock(
            this,
            root,
            name,
            id,
            initialState,
            initialError,
        );
    }
}
export {
    SupplierBlock,
    BoundSupplierBlock,
};
