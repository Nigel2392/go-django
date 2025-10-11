
import { Block, BlockMeta, BoundBlock } from '../base';
import { jsx } from '../../../../../admin/static_src/jsx';
import Icon from '../../../../../admin/static_src/utils/icon';
import { PanelComponent } from '../../../../../admin/static_src/utils/panels';
import { openAnimator } from '../../../../../admin/static_src/utils/animator';
import flash from '../../../../../admin/static_src/utils/flash';

type StreamBlockData = {
    id: string;
    type: string;
    order: number;
    data: any;
};

type StreamBlockValue = {
    blocks: StreamBlockData[];
};

type childBlockMap = {
    name: string;
    block: Block;
}[];

type BoundStreamBlockValue = {
    id: string;
    type: string;
    block: BoundBlock;    
};

class BoundStreamBlock extends BoundBlock<StreamBlock> {
    id: String;
    items: BoundStreamBlockValue[];
    itemWrapper: HTMLElement;
    totalInput: HTMLInputElement;
    activeItems: number;

    constructor(block: StreamBlock, root: HTMLElement, name: String, id: String, initialState: StreamBlockValue, initialError: any) {
        initialState = (initialState || { blocks: [] });
        initialError = (initialError || []);
        super(block, name, root);
        this.items = [];
        this.id = id;
        
        root.appendChild(
            <button type="button" data-action="add" class="stream-block-add-button" onClick={this._onAddClick.bind(this, null)}>
                { Icon('icon-plus', { title: window.i18n.gettext("Add %s", this.block.meta.label || window.i18n.gettext('item')) }) }
            </button>
        );

        this.totalInput = root.appendChild(
            <input type="hidden" data-stream-block-total name={ `${name}--total` } value={ String(initialState.blocks.length) }/>
        ) as HTMLInputElement;

        this.itemWrapper = root.appendChild(
            <div data-stream-block-items class="stream-block-items" data-sortable-target="items"></div>
        ) as HTMLElement;

        for (let i = 0; i < initialState.blocks.length; i++) {
            this._createChild(
                i, i, id, name, initialState.blocks[i], initialError[i] || null, false,
            );
        }
    }

    _createChild(suffix: number, sortIndex: number, id: String, name: String, value: StreamBlockData, error: any, animate: boolean = true) {
        var childBlock = this.block.childBlocks[value.type];
        if (!childBlock) {
            console.error(`No child block of type ${value.type} found in StreamBlock ${this.block.name}`);
            return;
        }

        if (!value.data) {
            value.data = this.block.defaults[value.type] || null;
        }

        let animator = null;
        if (animate) {
            animator = new openAnimator({
                duration: 300,
                onAdded: (elem) => {
                    elem.style.transition = "opacity 300ms ease";
                },
                onFinished: (elem) => {
                    elem.style.transition = "";
                    if (!elem.style) {
                        elem.removeAttribute("style");
                    }
                },
                animFrom: { opacity: "0" },
                animTo: { opacity: "1" },
            });
        }

        const itemId = `${id}-${suffix}`;
        const itemKey = `${name}-${suffix}`;
        const blockId = `${name}-id-${suffix}`;
        const orderId = `${name}-order-${suffix}`;
        const deletedKey = `${name}-${suffix}--deleted`;
        const typeKey = `${name}-${suffix}--type`;
        const panelId = `${itemKey}--panel`;
        const headingIndexId = `${itemKey}--heading-index`;
        const itemDom = (
            <div data-stream-block-field id={ itemKey + "--block" } data-index={ String(sortIndex) } data-sortable-target="item" data-replace={ `#${orderId};#${headingIndexId}+;[data-index]` } class="stream-block-field">
                <input type="hidden" id={ blockId } name={ blockId } value={ value.id } />
                <input type="hidden" id={ orderId } name={ orderId } value={ String(value.order) } />
                <input type="hidden" id={ deletedKey } name={ deletedKey } value=""/>
                <input type="hidden" id={ typeKey } name={ typeKey } value={ value.type } />

                {PanelComponent({
                    panelId: panelId,
                    heading: (
                        <div class="stream-block-field-heading">
                            { childBlock.meta.label ? <label for={ blockId } class="stream-block-field-heading-label">{ childBlock.meta.label }:</label> : null }
                            <span id={ headingIndexId } class="stream-block-field-heading-index">{ String(value.order + 1) }</span>
                        </div>
                    ),
                    children: (
                        <div data-stream-block-field-content class="stream-block-field-content">
                            <div data-stream-block-field-handle class="stream-block-field-drag-handle">
                                &#x2630;
                            </div>

                            <div data-stream-block-field-inner></div>

                            <div data-stream-block-field-actions class="stream-block-field-actions">
                                <div data-stream-block-field-actions-group class="stream-block-field-actions-group">
                                    <div data-stream-block-field-delete class="stream-block-field-delete">
                                        <button type="button" data-action="delete" class="stream-block-field-delete-button" onClick={this._onDeleteClick.bind(this, itemKey)}>
                                            { Icon('icon-trash') }
                                        </button>
                                    </div>
                                    <div data-stream-block-field-add class="stream-block-field-add">
                                        <button type="button" data-action="add" class="stream-block-field-add-button" onClick={this._onAddClick.bind(this, itemKey)}>
                                            { Icon('icon-plus') }
                                        </button>
                                    </div>
                                </div>
                            </div>
                        </div>
                    )
                })}
            </div>
        );
        
        if (animate) {
            animator.addElement(itemDom);
        }

        if (sortIndex === 0) { // append to start
            this.itemWrapper.prepend(itemDom);
        } else if (sortIndex >= this.items.length) { // append to end
            this.itemWrapper.appendChild(itemDom);
        } else { // insert in middle
            this.itemWrapper.insertBefore(itemDom, this.itemWrapper.children[sortIndex]);
        }

        const boundValue: BoundStreamBlockValue = {
            id: value.id,
            type: value.type,
            block: childBlock.render(
                itemDom.querySelector('[data-stream-block-field-inner]'),
                itemId,
                itemKey,
                value.data,
                error,
            )
        };
        
        this.items.push(boundValue);

        if (animate) {
            animator.start();
        }

        return itemDom;
    }

    _onDeleteClick(itemName: string, ev: MouseEvent) {
        ev.preventDefault();

        const wrapperId = `#${itemName}--block`;
        const elem = this.itemWrapper.querySelector(wrapperId) as HTMLElement;
        if (!elem) {
            console.warn("Couldn't find item to delete", wrapperId);
            return;
        }

        if (this.activeItems <= 1 && this.block.meta.required || (this.block.meta.minNum && this.activeItems <= this.block.meta.minNum)) {
            console.warn("Can't delete item, minimum reached");
            flash(elem);
            return;
        }
        
        const deletedInput = elem.querySelector(`#${itemName}--deleted`) as HTMLInputElement;
        if (!deletedInput) {
            console.warn("Couldn't find deleted input", `#${itemName}--deleted`);
            return;
        }

        elem.style.display = 'none';
        deletedInput.value = '1';
        this.activeItems -= 1;

        flash(this.itemWrapper, {
            color: 'orange',
            duration: 300,
            iters: 1,
            delay: 20,
        });
        
        const sortable = window.Stimulus.getControllerForElementAndIdentifier(this.element, "sortable");
        if (sortable) {
            (sortable as any).reOrderItems();
        } else {
            console.warn("Couldn't find sortable controller for list block", this.element);
        }
    }

    _onAddClick(itemName: string | null, ev: MouseEvent) {
        ev.preventDefault();

        if (this.block.meta.maxNum && this.activeItems >= this.block.meta.maxNum) {
            console.warn("Can't add item, maximum reached");
            flash(this.itemWrapper);
            return;
        }

        let index: number = 0;
        if (itemName) {
            itemName = `#${itemName}--block`;
            const elem = this.itemWrapper.querySelector(itemName) as HTMLElement;
            if (!elem) {
                console.warn("Couldn't find item to add", itemName);
                return;
            }
            index = parseInt(elem.dataset.index || '0', 10) + 1;
        }

    }
}

class StreamBlock extends Block {
    name: string;
    defaults: { [key: string]: any };
    childBlocks: { [key: string]: Block };

    constructor(name: string, childBlocks: childBlockMap, defaults: { [key: string]: any }, meta: BlockMeta) {
        super();
        this.name = name;
        this.meta = meta;
        this.childBlocks = {};
        this.defaults = defaults;
        for (let i = 0; i < childBlocks.length; i++) {
            this.childBlocks[childBlocks[i].name] = childBlocks[i].block;
        }
    }

    render(root: HTMLElement, id: String, name: String, initialState: any, initialError: any): any {
        return new BoundStreamBlock(
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
    StreamBlockValue,
    StreamBlock,
    BoundStreamBlock,
    BoundStreamBlockValue,
    StreamBlockData,
};
