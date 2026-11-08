import { Block, BlockMeta, BoundBlock } from '../base';
import { BoundSequenceBlock, BoundSequenceBlockValue } from './sequence-block';

type ListBlockValue = {
    id: string;
    order: number;
    data: any;
};

class BoundListBlockValue {
    id: string;
    order: number;
    block: BoundBlock;

    constructor(id: string, order: number, block: BoundBlock) {
        this.id = id;
        this.order = order;
        this.block = block;
    }

    getState() {
        return {
            id: this.id,
            order: this.order,
            data: this.block.getState(),
        }
    }
}

class BoundListBlock extends BoundSequenceBlock<ListBlock> {
    _getChildBlock(value: any): Block | undefined {
        return this.block.childBlock;
    }

    _renderBoundBlock(child: Block, placeholder: HTMLElement, id: string, name: String, value: any, error: any): BoundSequenceBlockValue {
        return new BoundListBlockValue(value.id, value.order, child.render(
            placeholder,
            id,
            name,
            value.data,
            error,
        ))
    }

    _defaultChildState(block: Block): any {
        return {
            id: '',
            data: block?.meta?.default || null,
        };
    }

    _onAddClick(itemName: string | null, ev: MouseEvent) {
        ev?.preventDefault();
        this._addBlock(itemName, this.block.childBlock, ev);
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