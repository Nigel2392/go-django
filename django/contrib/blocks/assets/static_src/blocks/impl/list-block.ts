
class ListBlock {
    constructor(items: any) {
        this.items = items;
    }

    items: any;
}

class ListBlockDef implements BlockDef {
    constructor(items: any) {
        this.items = items;
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