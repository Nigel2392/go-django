
import { BlockMeta, BoundBlock } from '../base';
import { jsx } from '../../../../../admin/static_src/jsx';
import Icon from '../../../../../admin/static_src/utils/icon';
import { PanelComponent } from '../../../../../admin/static_src/utils/panels';
import { openAnimator } from '../../../../../admin/static_src/utils/animator';
import flash from '../../../../../admin/static_src/utils/flash';
import { PanelElement } from '../../../../../admin/static_src/controllers/panel';
import { copyAttrs, slugify } from '../utils';

type BoundSequenceBlockValue = {
    id: string;
    block: BoundBlock<Block, HTMLElement>;
    [key: string]: any;
    getState(): any;
};

interface Block {
    meta: BlockMeta;
}

type SequenceBlockOptions = {
    allowSort: boolean;
    allowAdd: boolean;
    controllable: boolean;
    hideItemHeading?: boolean;
}

class BoundSequenceBlock<BLOCK extends Block = Block> extends BoundBlock<BLOCK, PanelElement> {
    id: String;
    options: SequenceBlockOptions;
    items: BoundSequenceBlockValue[];
    itemWrapper: HTMLElement;
    totalInput: HTMLInputElement;
    activeItems: number = 0;

    constructor(block: BLOCK, placeholder: HTMLElement, name: String, id: String, initialState: any, initialError: any, options: SequenceBlockOptions = {allowSort: true, allowAdd: true, controllable: true}) {
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

        let sortAttrs: any;
        if (options.allowSort) {
            sortAttrs = {
                "data-controller": "sortable",
                "data-sortable-draggable-value": ".sequence-block-field",
                // "data-sortable-handle-value": ".sequence-block-field-drag-handle",
            }
        }

        const root = PanelComponent({
            panelId: `panel-${slugify(name)}`,
            class: "sequence-block",
            allowPanelLink: !!block.meta.label,
            heading: block.meta.label && (!placeholder.dataset.isRootBlock) ? (
                <div class="sequence-block-field-heading">
                    <label for={id} class="sequence-block-field-heading-label">
                        {block.meta.label}:
                    </label>
                </div>
            ) : null,
            helpText: block.meta.helpText && (!placeholder.dataset.isRootBlock) ? (
                <div class="field-help">{block.meta.helpText}</div>
            ) : null,
            errors: errorsList,
            attrs: sortAttrs,
        });

        copyAttrs(
            placeholder,
            root.hasAttribute.bind(root),
            root.body.setAttribute.bind(root.body),
        );

        placeholder.replaceWith(root);

        super(block, name, root);
        this.options = options;
        this.items = [];
        this.id = id;
        
        if (options.allowAdd) {
            root.body.appendChild(
                <button type="button" class="sequence-block-add-button" aria-label={window.i18n.gettext("Add %s", block.meta.label || window.i18n.gettext('item'))} aria-expanded="false" onClick={this._onAddClick.bind(this, null)}>
                    { Icon('icon-plus', { title: window.i18n.gettext("Add %s", block.meta.label || window.i18n.gettext('item')) }) }
                </button>
            );
        }

        this.activeItems = initialState?.length || 0;
        this.totalInput = root.body.appendChild(
            <input type="hidden" data-sequence-block-total name={ `${name}--total` } value={ String(initialState?.length || 0) }/>
        ) as HTMLInputElement;

        this.itemWrapper = root.body.appendChild(
            <div data-sequence-block-items class="sequence-block-items" data-sortable-target="items"></div>
        ) as HTMLElement;

        for (let i = 0; i < (initialState?.length || 0); i++) {
            if (this.block.meta.maxNum && i > this.block.meta.maxNum) {
                break;
            }
            
            this._createChild(
                i, i, id, name, initialState[i], initialError?.errors?.[i]?.[0] || null, false, this._getChildBlock(initialState[i])!,
            );
        }
    }

    move(itemName: string, direction: 'up' | 'down', ev: MouseEvent) {
        ev?.preventDefault();
        const wrapperId = `#${itemName}--block`;
        const elem = this.itemWrapper.querySelector(wrapperId) as HTMLElement;
        if (!elem) {
            console.warn("Couldn't find item to move", wrapperId);
            return;
        }

        const sortable = window.Stimulus.getControllerForElementAndIdentifier(this.element, "sortable");
        if (sortable) {
            (sortable as any).moveItem(elem, direction);
        } else {
            console.warn("Couldn't find sortable controller for list block", this.element);
        }
    }

    _getChildBlock(value: any): Block | undefined {
        throw new Error("Not implemented");
    }

    _renderBoundBlock(child: Block, placeholder: HTMLElement, id: string, name: string, value: any, error: any): BoundSequenceBlockValue {
        throw new Error("Not implemented");
    }

    _defaultChildState(block: Block): any {
        throw new Error("Not implemented");
    }

    _getChildInputs(id: String, name: String, suffix: number, value: any): any {
        return null;
    }

    _onAddClick(itemName: string | null, ev: MouseEvent) {
        throw new Error("Not implemented");
    }

    _createChild(suffix: number, sortIndex: number, id: String, name: String, value: any, error: any, animate: boolean = true, child: Block): HTMLElement | undefined {
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
        let controls, actions, heading;

        if (!this.options.hideItemHeading) {
            heading = (
                <div class="sequence-block-field-heading">
                    { child.meta.label ? <label for={ blockId } class="sequence-block-field-heading-label">{ child.meta.label }:</label> : null }
                    <span id={ headingIndexId } class="sequence-block-field-heading-index">{ String(sortIndex + 1) }</span>
                </div>
            )
        }

        if (this.options.controllable) {
            controls = (
                <div data-sequence-block-field-controls class="sequence-block-field-controls">
                    <button type="button" class="sequence-block-field-move-up-button" aria-label={window.i18n.gettext("Move %s up", child.meta.label || window.i18n.gettext('item'))} onClick={this.move.bind(this, itemKey, 'up')}>
                        { Icon('icon-arrow-up', { title: window.i18n.gettext("Move %s up", child.meta.label || window.i18n.gettext('item')) }) }
                    </button>
                    <button type="button" class="sequence-block-field-move-down-button" aria-label={window.i18n.gettext("Move %s down", child.meta.label || window.i18n.gettext('item'))} onClick={this.move.bind(this, itemKey, 'down')}>
                        { Icon('icon-arrow-down', { title: window.i18n.gettext("Move %s down", child.meta.label || window.i18n.gettext('item')) }) }
                    </button>
                </div>
            )

            actions = (
                <div data-sequence-block-field-actions class="sequence-block-field-actions">
                    <div data-sequence-block-field-actions-group class="sequence-block-field-actions-group">
                        <div data-sequence-block-field-delete class="sequence-block-field-delete">
                            <button type="button" data-action="delete" class="sequence-block-field-delete-button sequence-block-field-actions-text" onClick={this._onDeleteClick.bind(this, itemKey)}>
                                { Icon('icon-trash') }
                            </button>
                        </div>
                        <div data-sequence-block-field-add class="sequence-block-field-add">
                            <button type="button" data-action="add" class="sequence-block-field-add-button sequence-block-field-actions-text" onClick={this._onAddClick.bind(this, itemKey)}>
                                { Icon('icon-plus') }
                            </button>
                        </div>
                    </div>
                </div>
            )
        }

        const itemDom = (
            <div data-sequence-block-field id={ itemKey + "--block" } data-index={ String(sortIndex) } data-sortable-target="item" data-replace={ `#${orderId};#${headingIndexId}+;[data-index]` } class="sequence-block-field">
                <input type="hidden" id={ blockId } name={ blockId } value={ value.id } />
                <input type="hidden" id={ orderId } name={ orderId } value={ String(sortIndex) } />
                <input type="hidden" id={ deletedKey } name={ deletedKey } value=""/>
                { this._getChildInputs(id, name, suffix, value) }

                {PanelComponent({
                    panelId: panelId,
                    heading: heading,
                    children: this._getChildPanelContentPlaceholder(controls, actions)
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

        const boundValue: BoundSequenceBlockValue = this._renderBoundBlock(
            child,
            itemDom.querySelector('[data-sequence-block-field-placeholder]'),
            itemId,
            itemKey,
            value,
            error,
        );

        this.items.push(boundValue);

        if (animate) {
            animator.start();
        }

        return itemDom;
    }

    getState(): any{
        let state = [];
        for (let i = 0; i < this.items.length; i++) {
            state.push(this.items[i].getState())
        }
        return state;
    }

    _getChildPanelContentPlaceholder(controls: HTMLElement, actions: HTMLElement): HTMLElement {
        return (
            <div data-sequence-block-field-content class="sequence-block-field-content">
                { controls }

                <div data-sequence-block-field-placeholder></div>

                { actions }
            </div>
        )
    }

    _onDeleteClick(itemName: string, ev: MouseEvent) {
        ev?.preventDefault();

        this.items = this.items.filter((value) => {
            return value.id != itemName
        })

        const wrapperId = `#${itemName}--block`;
        const elem = this.itemWrapper.querySelector(wrapperId) as HTMLElement;
        if (!elem) {
            console.warn("Couldn't find item to delete", wrapperId);
            return;
        }

        if (this.activeItems <= 1 && this.block.meta.required || (this.block.meta.minNum && this.block.meta.minNum > 0 && this.activeItems <= this.block.meta.minNum)) {
            console.warn("Can't delete item, minimum reached");
            flash(elem);
            return;
        }
        
        const deletedInput = elem.querySelector(`#${itemName}--deleted`) as HTMLInputElement;
        if (!deletedInput) {
            console.warn("Couldn't find deleted input", `#${itemName}--deleted`);
            return;
        }

        elem.querySelectorAll('input, textarea, select, fieldset').forEach((el: HTMLElement) => {
            const input = el as HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement;
            input.disabled = true;
            input.setAttribute('disabled', '');
        });

        deletedInput.disabled = false;
        deletedInput.removeAttribute('disabled');

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

    _addBlock(itemName: string | null, block: Block, ev: MouseEvent) {
        ev?.preventDefault();

        let replaceTarget: HTMLElement | null = null;
        let replaceIndex: number | null = null;

        if (this.block.meta.maxNum && this.block.meta.maxNum > 0 && this.activeItems >= this.block.meta.maxNum) {
            if (!(this.block.meta.minNum && this.block.meta.minNum === this.block.meta.maxNum)) {
                flash(this.itemWrapper);
                return;
            }

            // REPLACE IN PLACE
            if (itemName) {
                replaceTarget = this.itemWrapper.querySelector(`#${itemName}--block`) as HTMLElement;
            } else {
                replaceTarget = this.itemWrapper.querySelector('.sequence-block-field:not([style*="display: none"])') as HTMLElement;
            }

            if (!replaceTarget) {
                console.warn("Couldn't find item to replace");
                flash(this.itemWrapper);
                return;
            }

            replaceIndex = parseInt(replaceTarget.dataset.index || '0', 10);
            const targetItemName = replaceTarget.id.replace(/--block$/, '');
            const deletedInput = replaceTarget.querySelector(`#${targetItemName}--deleted`) as HTMLInputElement
                || replaceTarget.querySelector(`input[name$="--deleted"]`) as HTMLInputElement;

            replaceTarget.querySelectorAll('input, textarea, select, fieldset').forEach((el: HTMLElement) => {
                const input = el as HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement;
                input.disabled = true;
                input.setAttribute('disabled', '');
            });

            if (deletedInput) {
                deletedInput.disabled = false;
                deletedInput.removeAttribute('disabled');
                deletedInput.value = '1';
            }

            this.items = this.items.filter((value) => {
                return value.id != targetItemName
            })

            replaceTarget.style.display = 'none';
            this.activeItems -= 1;
        }

        let index: number = 0;
        if (replaceIndex !== null) {
            index = replaceIndex;
        } else if (itemName) {
            const selector = `#${itemName}--block`;
            const elem = this.itemWrapper.querySelector(selector) as HTMLElement;
            if (!elem) {
                console.warn("Couldn't find item to add", selector);
                return;
            }
            index = parseInt(elem.dataset.index || '0', 10) + 1;
        } else {
            index = this.activeItems;
        }

        const newValue = this._defaultChildState(block);
        const childDom = this._createChild(
            this.items.length, index, this.id, this.name, newValue, null, true, block,
        );

        if (childDom) {
            if (replaceTarget) {
                replaceTarget.after(childDom);
            }

            flash(childDom, {
                color: 'green',
                duration: 300,
                iters: 1,
                delay: 100,
            });
        }

        this.totalInput.value = String(this.items.length);
        this.activeItems += 1;

        const sortable = window.Stimulus.getControllerForElementAndIdentifier(this.element, "sortable");
        if (sortable) {
            (sortable as any).reOrderItems();
        }
    }
}

export {
    BoundSequenceBlockValue,
    BoundSequenceBlock,
};