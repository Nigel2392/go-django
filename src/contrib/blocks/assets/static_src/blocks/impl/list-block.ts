import { Block, BoundBlock, Config } from '../base';

class ListBlockValue {
    id: string;
    order: number;
    data: any;

    constructor(id: string, order: number, data: any) {
        this.id = id;
        this.order = order;
        this.data = data;
    }
}

class BoundListBlock extends BoundBlock {
    constructor(block: Block, root: HTMLElement, prefix: String, initialState: any, initialError: any) {
        super(block, prefix);
    }

    items: any;
}

type ListBlockElementConfig = Config & {
    minNum?: number;
    maxNum?: number;
};

type ListBlockConfig = Config & {
    childBlock: Block;
    element: ListBlockElementConfig;
}

class ListBlock extends Block<ListBlockConfig> {
    constructor(config: ListBlockConfig) {
        super(config);
    }

    render(root: HTMLElement, name: String, initialState: any, initialError: any): any {
        return new BoundListBlock(
            this,
            root,
            name,
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