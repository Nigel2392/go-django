import { Block, BlockDef, Config } from '../base';

class ListBlock {
    constructor(items: any) {
        this.items = items;
    }

    items: any;
}

class ListBlockDef extends BlockDef {

    constructor(element: HTMLElement, config: Config) {
        super(element, config);
        console.log("ListBlockDef constructor", element, config);
    }

    render(): any {
        return new ListBlock(this.items);
    }

    items: any;
}

export {
    ListBlock,
    ListBlockDef,
};