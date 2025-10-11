
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

class BoundStreamBlock extends BoundBlock<StreamBlock> {
    id: String;
    items: BoundBlock[];
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
    }

    _onDeleteClick(itemName: string, ev: MouseEvent) {
        ev.preventDefault();
    }
    _onAddClick(itemName: string | null, ev: MouseEvent) {
        ev.preventDefault();
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
};
