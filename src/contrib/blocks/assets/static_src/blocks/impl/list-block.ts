import { Block, BlockDef, Config, ConfigBlock, ConfigElement } from '../base';

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

class ListBlock {
    constructor(items: any) {
        this.items = items;
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
    childBlock: BlockDef;
    element: ListBlockElement;
}

class ListBlockDef extends BlockDef<ListBlockConfig> {

    constructor(element: HTMLElement, config: ListBlockConfig) {
        super(element, config);
    }

    render(placeholder: HTMLElement, prefix: String, initialState: any, initialError: any): any {
        console.log("ListBlockDef render", placeholder, prefix, initialState, initialError, this);
        return new ListBlock(this.items);
    }

    items: any;
}

export {
    ListBlockValue,
    ListBlock,
    ListBlockDef,
};