
import { Block, BoundBlock } from '../base';
import { jsx } from '../../../../../admin/static_src/jsx';
import Icon from '../../../../../admin/static_src/utils/icon';
import { PanelComponent } from '../../../../../admin/static_src/utils/panels';
import { openAnimator } from '../../../../../admin/static_src/utils/animator';
import flash from '../../../../../admin/static_src/utils/flash';
import { PanelElement } from '../../../../../admin/static_src/controllers/panel';
import { copyAttrs } from '../utils';

type BoundSequenceBlockValue = {
    id: string;
    block: BoundBlock<Block, HTMLElement>;
    [key: string]: any;
};

class BoundSequenceBlock extends BoundBlock<any, PanelElement> {
    id: String;
    items: BoundSequenceBlockValue[];
    itemWrapper: HTMLElement;
    totalInput: HTMLInputElement;
    activeItems: number;
    extra: (id: string, name: string, suffix: string, value: any) => void;

    constructor(block: Block, placeholder: HTMLElement, name: String, id: String, initialState: any, initialError: any) {
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
            helpText: block.meta.helpText ? (
                <div class="field-help">{block.meta.helpText}</div>
            ) : null,
            errors: errorsList,
            attrs: {
                "data-controller": "sortable",
            },
        });

        copyAttrs(
            placeholder,
            root.hasAttribute.bind(root),
            root.body.setAttribute.bind(root.body),
        );

        placeholder.replaceWith(root);

        super(block, name, root);
        this.items = [];
        this.id = id;
        
        root.body.appendChild(
            <button type="button" class="sequence-block-add-button" aria-label={window.i18n.gettext("Add %s", block.meta.label || window.i18n.gettext('item'))} aria-expanded="false" onClick={this._onAddClick.bind(this, null)}>
                { Icon('icon-plus', { title: window.i18n.gettext("Add %s", block.meta.label || window.i18n.gettext('item')) }) }
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
        const itemDom = (
            <div data-sequence-block-field id={ itemKey + "--block" } data-index={ String(sortIndex) } data-sortable-target="item" data-replace={ `#${orderId};#${headingIndexId}+;[data-index]` } class="sequence-block-field">
                <input type="hidden" id={ blockId } name={ blockId } value={ value.id } />
                <input type="hidden" id={ orderId } name={ orderId } value={ String(sortIndex) } />
                <input type="hidden" id={ deletedKey } name={ deletedKey } value=""/>
                { this._getChildInputs(id, name, suffix, value) }

                {PanelComponent({
                    panelId: panelId,
                    heading: (
                        <div class="sequence-block-field-heading">
                            { child.meta.label ? <label for={ blockId } class="sequence-block-field-heading-label">{ child.meta.label }:</label> : null }
                            <span id={ headingIndexId } class="sequence-block-field-heading-index">{ String(sortIndex + 1) }</span>
                        </div>
                    ),
                    children: (
                        <div data-sequence-block-field-content class="sequence-block-field-content">
                            <div data-sequence-block-field-controls class="sequence-block-field-controls">
                                <button type="button" class="sequence-block-field-move-up-button" aria-label={window.i18n.gettext("Move %s up", child.meta.label || window.i18n.gettext('item'))} onClick={this.move.bind(this, itemKey, 'up')}>
                                    { Icon('icon-arrow-up', { title: window.i18n.gettext("Move %s up", child.meta.label || window.i18n.gettext('item')) }) }
                                </button>
                                <button type="button" class="sequence-block-field-move-down-button" aria-label={window.i18n.gettext("Move %s down", child.meta.label || window.i18n.gettext('item'))} onClick={this.move.bind(this, itemKey, 'down')}>
                                    { Icon('icon-arrow-down', { title: window.i18n.gettext("Move %s down", child.meta.label || window.i18n.gettext('item')) }) }
                                </button>
                                <button type="button" class="sequence-block-field-drag-handle" aria-label={window.i18n.gettext("Drag %s to reorder", child.meta.label || window.i18n.gettext('item'))}>
                                    { Icon('icon-grip-horizontal', { title: window.i18n.gettext("Drag %s to reorder", child.meta.label || window.i18n.gettext('item')) }) }
                                </button>
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
                                        <button type="button" data-action="add" class="sequence-block-field-add-button sequence-block-field-actions-text" onClick={this._onAddClick.bind(this, itemKey)}>
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

    _addBlock(itemName: string | null, block: Block, ev: MouseEvent) {
        ev?.preventDefault();

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


        const newValue = this._defaultChildState(block);
        flash(this._createChild(
            this.items.length, index, this.id, this.name, newValue, null, true, block,
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

export {
    BoundSequenceBlockValue,
    BoundSequenceBlock,
};