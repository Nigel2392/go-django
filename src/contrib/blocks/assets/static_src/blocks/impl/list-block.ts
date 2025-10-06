import { Block, BoundBlock, Config, ConfigBlock, ConfigElement } from '../base';

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
    constructor(blockDef: Block, prefix: String, items: any) {
        super(blockDef, prefix);
    }

    items: any;
}

type ListBlockElementConfig = ConfigElement & {
    minNum?: number;
    maxNum?: number;
};

type ListBlockElement = ConfigBlock & {
    element: ListBlockElementConfig;
}

type ListBlockConfig = Config & {
    childBlock: Block;
    element: ListBlockElement;
}

class ListBlock extends Block<ListBlockConfig> {

    constructor(element: HTMLElement, config: ListBlockConfig) {
        super(element, config);
    }

    render(placeholder: HTMLElement, prefix: String, initialState: any, initialError: any): any {
        return new BoundListBlock(this, prefix, initialState);
    }

    items: any;
}

export {
    ListBlockValue,
    ListBlock,
    BoundListBlock,
};