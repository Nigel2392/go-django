
class FieldBlock {
    constructor(config: any) {
        this.config = config;
    }

    config: any;
}

class FieldBlockDef implements BlockDef {
    constructor(config: any) {
        this.config = config;
    }

    render(): any {
        console.log(this.config);
        return new FieldBlock(this.config);
    }

    config: any;
}

export {
    FieldBlock,
    FieldBlockDef,
};