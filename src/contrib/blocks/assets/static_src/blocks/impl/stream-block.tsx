
import { Block, BlockMeta, BoundBlock } from '../base';
import { jsx } from '../../../../../admin/static_src/jsx';
import Icon from '../../../../../admin/static_src/utils/icon';
import { PanelComponent } from '../../../../../admin/static_src/utils/panels';
import { openAnimator } from '../../../../../admin/static_src/utils/animator';
import flash from '../../../../../admin/static_src/utils/flash';
import { PanelElement } from '../../../../../admin/static_src/controllers/panel';
import tippy, { Instance as TippyInstance } from 'tippy.js';

type StreamBlockData = {
    id: string;
    type: string;
    data: any;
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

class BoundStreamBlock extends BoundBlock<StreamBlock, PanelElement> {
    id: String;
    items: BoundStreamBlockValue[];
    itemWrapper: HTMLElement;
    totalInput: HTMLInputElement;
    dropdown: TippyInstance;
    activeItems: number;

    constructor(block: StreamBlock, placeholder: HTMLElement, name: String, id: String, initialState: StreamBlockData[], initialError: any) {
        initialState = (initialState || []);
        initialError = (initialError || {});

        let errorsList = null;
        if (initialError && initialError?.nonBlockErrors) {
            errorsList = (
                <ul class="field-errors">
                    {initialError?.nonBlockErrors?.map((err: string) => (
                        <li class="field-error">{err}</li>
                    ))}
                </ul>
            );
        }

        const root = PanelComponent({
            panelId: `${name}--panel`,
            class: "sequence-block",
            allowPanelLink: !!block.meta.label,
            heading: block.meta.label ? (
                <div class="sequence-block-field-heading">
                    <label for={id} class="sequence-block-field-heading-label">
                        {block.meta.label}:
                    </label>
                </div>
            ) : null,
            errors: errorsList,
            attrs: {
                "data-controller": "sortable",
            },
        });

        placeholder.replaceWith(root);

        super(block, name, root);
        this.items = [];
        this.id = id;
        
        root.body.appendChild(
            <button type="button" class="sequence-block-add-button" aria-label={window.i18n.gettext("Add %s", this.block.meta.label || window.i18n.gettext('item'))} aria-expanded="false" onClick={this._chooseTypeClick.bind(this, null)}>
                { Icon('icon-plus', { title: window.i18n.gettext("Add %s", this.block.meta.label || window.i18n.gettext('item')) }) }
            </button>
        );

        this.totalInput = root.body.appendChild(
            <input type="hidden" data-sequence-block-total name={ `${name}--total` } value={ String(initialState?.length || 0) }/>
        ) as HTMLInputElement;

        this.itemWrapper = root.body.appendChild(
            <div data-sequence-block-items class="sequence-block-items" data-sortable-target="items"></div>
        ) as HTMLElement;

        for (let i = 0; i < (initialState?.length || 0); i++) {
            this._createChild(
                i, i, id, name, initialState[i], initialError?.errors?.[i]?.[0] || null, false,
            );
        }
    }

    _createChild(suffix: number, sortIndex: number, id: String, name: String, value: StreamBlockData, error: any, animate: boolean = true) {
        var childBlock = this.block.childBlocks.get(value.type);
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
            <div data-sequence-block-field id={ itemKey + "--block" } data-index={ String(sortIndex) } data-sortable-target="item" data-replace={ `#${orderId};#${headingIndexId}+;[data-index]` } class="sequence-block-field">
                <input type="hidden" id={ blockId } name={ blockId } value={ value.id } />
                <input type="hidden" id={ orderId } name={ orderId } value={ String(sortIndex) } />
                <input type="hidden" id={ deletedKey } name={ deletedKey } value=""/>
                <input type="hidden" id={ typeKey } name={ typeKey } value={ value.type } />

                {PanelComponent({
                    panelId: panelId,
                    heading: (
                        <div class="sequence-block-field-heading">
                            { childBlock.meta.label ? <label for={ blockId } class="sequence-block-field-heading-label">{ childBlock.meta.label }:</label> : null }
                            <span id={ headingIndexId } class="sequence-block-field-heading-index">{ String(sortIndex + 1) }</span>
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
                                        <button type="button" data-action="delete" class="sequence-block-field-delete-button sequence-block-field-actions-text" onClick={this._onDeleteClick.bind(this, itemKey)}>
                                            { Icon('icon-trash') }
                                        </button>
                                    </div>
                                    <div data-sequence-block-field-add class="sequence-block-field-add">
                                        <button type="button" data-action="add" class="sequence-block-field-add-button sequence-block-field-actions-text" onClick={this._chooseTypeClick.bind(this, itemKey)}>
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
                itemDom.querySelector('[data-sequence-block-field-placeholder]'),
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
        ev?.preventDefault();

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

    _chooseTypeClick(itemName: string | null, ev: MouseEvent) {
        ev?.preventDefault();

        if (this.block.childBlocks.size === 1) {
            const onlyType = Array.from(this.block.childBlocks.keys())[0];
            this._onAddClick(itemName, onlyType, ev);
            return;
        }

        this.dropdown?.destroy();
        this.dropdown = tippy(ev.currentTarget as HTMLElement, {
            content: this._renderTypeMenu(itemName),
            allowHTML: true,
            interactive: true,
            trigger: 'manual',

            theme: 'dropdown',
            arrow: true,
            maxWidth: 350,
            placement: 'bottom',
        })

        this.dropdown.show();
    }

    _renderTypeMenu(itemName: string | null) {
        return (
            <div class="dropdown">
                <div class="sequence-block-add-dropdown dropdown__content">
                    { Array.from(this.block.childBlocks.entries()).map(([type, block]) => (
                        <button type="button" class="sequence-block-add-type-button" onClick={this._onAddClick.bind(this, itemName, type)} data-block-type={type}>
                            { block.meta.label || type }
                        </button>
                    )) }
                </div>
            </div>
        )
    }

    _onAddClick(itemName: string | null, typeName: string, ev: MouseEvent) {
        ev?.preventDefault();

        // Check if control key is pressed to allow multiple additions
        const isCtrlPressed = ev.ctrlKey || ev.metaKey;
        if (!isCtrlPressed) {
            this.dropdown?.destroy();
            this.dropdown = null;
        }

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

        const newValue: StreamBlockData = {
            id: '',
            type: typeName,
            data: null,
        };
        if (this.block.defaults[typeName]) {
            newValue.data = this.block.defaults[typeName];
        }
        flash(this._createChild(
            this.items.length, index, this.id, this.name, newValue, null,
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

class StreamBlock extends Block {
    name: string;
    defaults: { [key: string]: any };
    childBlocks: Map<string, Block>;

    constructor(name: string, childBlocks: childBlockMap, defaults: { [key: string]: any }, meta: BlockMeta) {
        super();
        this.name = name;
        this.meta = meta;
        this.childBlocks = new Map<string, Block>();
        this.defaults = defaults;
        for (let i = 0; i < childBlocks.length; i++) {
            this.childBlocks.set(childBlocks[i].name, childBlocks[i].block);
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
    StreamBlockData,
    StreamBlock,
    BoundStreamBlock,
    BoundStreamBlockValue,
};
