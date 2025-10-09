import { Block, BlockMeta, BoundBlock } from '../base';
import { jsx } from '../../../../../admin/static_src/jsx';

type ListBlockValue = {
    id: string;
    order: number;
    data: any;
};

class BoundListBlock extends BoundBlock {
    id: String;
    items: BoundBlock[];
    itemWrapper: HTMLElement;
    totalInput: HTMLInputElement;

    constructor(block: ListBlock, root: HTMLElement, name: String, id: String, initialState: any, initialError: any) {
        initialState = initialState || [];
        initialError = initialError || [];

        super(block, name, root);
        this.items = [];
        this.id = id;

        this.itemWrapper = (
            <div data-list-block-items class="list-block-items" data-sortable-target="items"></div>
        );

        this.totalInput = (
            <input type="hidden" data-list-block-total name={ `${name}--total` } value={ String(initialState.length) }/>
        ) as HTMLInputElement;

        root.appendChild(this.totalInput);
        root.appendChild(this.itemWrapper);

        for (let i = 0; i < initialState.length; i++) {
            this._createChild(
                i, i, id, name, initialState[i], initialError[i] || null,
            )
        }
    }

    _createChild(suffix: number, sortIndex: number, id: String, name: String, value: ListBlockValue, error: any) {
        const itemId = `${id}-${suffix}`;
        const itemKey = `${name}-${suffix}`;
        const blockId = `${name}-id-${suffix}`;
        const orderId = `${name}-order-${suffix}`;
        const deletedKey = `${name}-${suffix}--deleted`;
        const itemDom = (
            <div data-list-block-field id={ itemKey + "--block" } data-index={ String(sortIndex) } data-sortable-target="item" data-replace={ `#${orderId};[data-index]` } class="list-block-field">
                <input type="hidden" id={ blockId } name={ blockId } value={ value.id } />
                <input type="hidden" id={ orderId } name={ orderId } value={ String(value.order) } />
                <input type="hidden" id={ deletedKey } name={ deletedKey } value=""/>

                <div data-list-block-field-label class="list-block-field-label">
                    { this.block.meta.label ? <label for={ blockId }>{ this.block.meta.label }</label> : null }
                </div>

                <div data-list-block-field-content class="list-block-field-content">
                    <div data-list-block-field-handle class="list-block-field-drag-handle">
                        &#x2630;
                    </div>
                    <div data-list-block-field-inner></div>
                </div>

                <div data-list-block-field-delete class="list-block-field-delete">
                    <button type="button" data-action="delete" class="list-block-field-delete-button" onClick={this._onDeleteClick.bind(this, itemKey)}> - </button>
                </div>
                <div data-list-block-field-add class="list-block-field-add">
                    <button type="button" data-action="add" class="list-block-field-add-button" onClick={this._onAddClick.bind(this, itemKey)}>+</button>
                </div>
            </div>
        );

        this.itemWrapper.appendChild(itemDom);
        this.items.push(this.block.childBlock.render(
            itemDom.querySelector('[data-list-block-field-inner]'),
            itemId,
            itemKey,
            value.data,
            error,
        ));
    }

    _onDeleteClick(itemName: string, ev: MouseEvent) {
        ev.preventDefault();
        const wrapperId = `#${itemName}--block`;
        const elem = this.itemWrapper.querySelector(wrapperId) as HTMLElement;
        if (!elem) {
            console.warn("Couldn't find item to delete", wrapperId);
            return;
        }

        const deletedInput = elem.querySelector(`#${itemName}--deleted`) as HTMLInputElement;
        if (!deletedInput) {
            console.warn("Couldn't find deleted input", `#${itemName}--deleted`);
            return;
        }

        elem.style.display = 'none';
        deletedInput.value = '1';
    }

    _onAddClick(itemName: string, ev: MouseEvent) {
        ev.preventDefault();
        itemName = `#${itemName}--block`;
        const elem = this.itemWrapper.querySelector(itemName) as HTMLElement;
        if (!elem) {
            console.warn("Couldn't find item to add", itemName);
            return;
        }
        const index = parseInt(elem.dataset.index || '0', 10) + 1;
        this._createChild(
            this.items.length, index, this.id, this.name, { id: '', order: index, data: null }, null,
        );
        this.totalInput.value = String(this.items.length);
    }
}

class ListBlock extends Block {
    name: string;
    childBlock: Block;
    
    constructor(name: string, childBlock: Block, meta: BlockMeta) {
        super();
        this.name = name;
        this.childBlock = childBlock;
        this.meta = meta;
    }

    render(root: HTMLElement, id: String, name: String, initialState: any, initialError: any): any {
        return new BoundListBlock(
            this,
            root,
            name,
            id,
            initialState,
            initialError,
        );
    }

    items: any;
}

export {
    ListBlockValue,
    ListBlock,
    BoundListBlock,
};