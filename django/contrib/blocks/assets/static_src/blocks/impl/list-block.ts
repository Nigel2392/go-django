import { Block, BlockDef, Config } from '../base';

class ListBlock {
    constructor(items: any) {
        this.items = items;
    }

    items: any;
}

class ListBlockDef extends BlockDef {

    render(): any {
        return new ListBlock(this.items);
    }

    items: any;
}

export {
    ListBlock,
    ListBlockDef,
};