import { Block, BlockMeta, BoundBlock } from '../base';
import { jsx } from '../../../../../editor/features/links/static_src/jsx';

type ListBlockValue = {
    id: string;
    order: number;
    data: any;
};

        //<div data-list-block data-controller="sortable block" class="list-block" data-block-class-path-value="django.blocks.list-block" data-block-class-args-value={ telepath } data-block-id-value={ id } data-block-arguments-value={ templ.JSONString([]any{name, valueArr, listBlockErrors}) }>
        //
	    //	<input data-list-block-add type="hidden" name={ fmt.Sprintf("%s-added", name) } value={ strconv.Itoa(len(valueArr)) }>
        //
	    //	{{ var iStr string }}
        //
	    //	<div data-list-block-items data-sortable-target="items" class="list-block-items">
	    //		for i, v := range valueArr {
                //
	    //    		{{ var id  = fmt.Sprintf("%s-%d", id, i) }}
        //            {{ var blockId = fmt.Sprintf("%s-id-%d", name, i) }}
        //            {{ var orderId = fmt.Sprintf("%s-order-%d", name, i) }}
	    //    		{{ var key = fmt.Sprintf("%s-%d", name, i) }}
                //
	    //			{{ iStr = strconv.Itoa(i) }}
                //
        //    	    <div data-list-block-field data-index={ iStr } data-sortable-target="item" data-replace={ fmt.Sprintf("#%s;[data-index]", orderId) } class="list-block-field">
                    //
        //                <input type="hidden" id={ blockId } name={ blockId } value={ v.ID.String() }>
        //                <input type="hidden" id={ orderId } name={ orderId } value={ strconv.Itoa(v.Order) }>
        //                {{ var newErrs = listBlockErrors.Get(i) }}
        //                <div data-list-block-field-content>
        //                    @renderForm(ctx, l.Child, id, key, v.Data, newErrs, tplCtx)
        //                </div>
        //    	    </div>
	    //    	}
	    //	</div>
        //</div>


class BoundListBlock extends BoundBlock {
    items: BoundBlock[];
    itemWrapper: HTMLElement;
    addedInput: HTMLInputElement;

    constructor(block: ListBlock, root: HTMLElement, name: String, id: String, initialState: any, initialError: any) {
        initialState = initialState || [];
        initialError = initialError || [];

        super(block, name, root);
        this.items = [];

        this.itemWrapper = (
            <div data-list-block-items class="list-block-items" data-sortable-target="items"></div>
        );

        this.addedInput = (
             <input type="hidden" data-list-block-add name={ `${name}-added` } value={ String(initialState.length) }/>
        ) as HTMLInputElement;

        root.appendChild(this.addedInput);
        root.appendChild(this.itemWrapper);

        for (let i = 0; i < initialState.length; i++) {
            this._createChild(
                i, id, name, initialState[i], initialError[i] || null,
            )
        }
    }

    _createChild(index: number, id: String, name: String, value: ListBlockValue, error: any) {
        const itemId = `${id}-${index}`;
        const itemKey = `${name}-${index}`;
        const blockId = `${name}-id-${index}`;
        const orderId = `${name}-order-${index}`;
        const itemDom = (
            <div data-list-block-field data-index={ String(index) } data-sortable-target="item" data-replace={ `#${orderId};[data-index]` } class="list-block-field">
                <input type="hidden" id={ blockId } name={ blockId } value={ value.id } />
                <input type="hidden" id={ orderId } name={ orderId } value={ String(value.order) } />

                <div data-list-block-field-label class="list-block-field-label">
                    { this.block.meta.label ? <label for={ blockId }>{ this.block.meta.label }</label> : null }
                </div>

                <div data-list-block-field-content class="list-block-field-content">
                    <div data-list-block-field-handle class="list-block-field-drag-handle">
                        &#x2630;
                    </div>
                    <div data-list-block-field-inner></div>
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