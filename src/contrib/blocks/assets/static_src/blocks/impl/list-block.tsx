import { Block, BlockMeta, BoundBlock } from '../base';
import { jsx } from '../../../../../admin/static_src/jsx';
import Icon from '../../../../../admin/static_src/utils/icon';
import { Panel, PanelComponent } from '../../../../../admin/static_src/utils/panels';
import { openAnimator } from '../../../../../admin/static_src/utils/animator';
import flash from '../../../../../admin/static_src/utils/flash';

type ListBlockValue = {
    id: string;
    order: number;
    data: any;
};

class BoundListBlock extends BoundBlock<ListBlock, Panel> {
    id: String;
    items: BoundBlock[];
    itemWrapper: HTMLElement;
    totalInput: HTMLInputElement;
    activeItems: number;

    constructor(block: ListBlock, placeholder: HTMLElement, name: String, id: String, initialState: any, initialError: any) {
        initialState = initialState || [];
        initialError = initialError || [];

        const root = PanelComponent({
            panelId: `${name}--panel`,
            class: "sequence-block",
            allowPanelLink: true,
            heading: block.meta.label ?? (
                <div class="sequence-block-field-heading">
                    <label for={id} class="sequence-block-field-heading-label">{block.meta.label}:</label>
                </div>
            ),
            attrs: {
                "data-controller": "sortable",
            },
        });

        placeholder.replaceWith(root);

        super(block, name, root);
        this.items = [];
        this.id = id;

        root.body.appendChild(
            <button type="button" data-action="add" class="sequence-block-add-button" onClick={this._onAddClick.bind(this, null)}>
                { Icon('icon-plus', { title: window.i18n.gettext("Add %s", this.block.meta.label || window.i18n.gettext('item')) }) }
            </button>
        );

        this.totalInput = root.body.appendChild(
            <input type="hidden" data-sequence-block-total name={ `${name}--total` } value={ String(initialState.length) }/>
        ) as HTMLInputElement;

        this.itemWrapper = root.body.appendChild(
            <div data-sequence-block-items class="sequence-block-items" data-sortable-target="items"></div>
        ) as HTMLElement;


        for (let i = 0; i < initialState.length; i++) {
            this._createChild(
                i, i, id, name, initialState[i], initialError[i] || null, false,
            )
        }

        this.activeItems = this.items.length;
    }

    _createChild(suffix: number, sortIndex: number, id: String, name: String, value: ListBlockValue, error: any, animate: boolean = true) {
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
        const panelId = `${itemKey}--panel`;
        const headingIndexId = `${itemKey}--heading-index`;
        const itemDom = (
            <div data-sequence-block-field id={ itemKey + "--block" } data-index={ String(sortIndex) } data-sortable-target="item" data-replace={ `#${orderId};#${headingIndexId}+;[data-index]` } class="sequence-block-field">
                <input type="hidden" id={ blockId } name={ blockId } value={ value.id } />
                <input type="hidden" id={ orderId } name={ orderId } value={ String(value.order) } />
                <input type="hidden" id={ deletedKey } name={ deletedKey } value=""/>

                {PanelComponent({
                    panelId: panelId,
                    heading: (
                        <div class="sequence-block-field-heading">
                            { this.block.meta.label ? <label for={ blockId } class="sequence-block-field-heading-label">{ this.block.meta.label }:</label> : null }
                            <span id={ headingIndexId } class="sequence-block-field-heading-index">{ String(value.order + 1) }</span>
                        </div>
                    ),
                    children: (
                        <div data-sequence-block-field-content class="sequence-block-field-content">
                            <div data-sequence-block-field-handle class="sequence-block-field-drag-handle">
                                &#x2630;
                            </div>

                            <div data-sequence-block-field-placeholder></div>

                            <div data-sequence-block-field-actions class="sequence-block-field-actions">
                                <div data-sequence-block-field-actions-group class="sequence-block-field-actions-group">
                                    <div data-sequence-block-field-delete class="sequence-block-field-delete">
                                        <button type="button" data-action="delete" class="sequence-block-field-delete-button" onClick={this._onDeleteClick.bind(this, itemKey)}>
                                            { Icon('icon-trash') }
                                        </button>
                                    </div>
                                    <div data-sequence-block-field-add class="sequence-block-field-add">
                                        <button type="button" data-action="add" class="sequence-block-field-add-button" onClick={this._onAddClick.bind(this, itemKey)}>
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
            
        this.items.push(this.block.childBlock.render(
            itemDom.querySelector('[data-sequence-block-field-placeholder]'),
            itemId,
            itemKey,
            value.data,
            error,
        ));

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

        flash(this._createChild(
            this.items.length, index, this.id, this.name, { id: '', order: index, data: null }, null,
        ), {
            color: 'green',
            duration: 300,
            iters: 1,
            delay: 100,
        });

        this.totalInput.value = String(this.items.length);
        this.activeItems += 1;
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
}

export {
    ListBlockValue,
    ListBlock,
    BoundListBlock,
};