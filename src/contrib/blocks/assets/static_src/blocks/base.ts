import { Widget } from "../widgets/widget";

//type Block = {
//    widget: HTMLElement;
//    blockDef: BlockDef;
//
//    getLabel(): string;
//    getState(): any;
//    getError(): any;
//    setState(state: any): void;
//    setError(errors: any): void;
//}

type BlockMeta = {
    label?: string;
    helpText?: string;
    required?: boolean;
    default?: any;
    attrs?: { [key: string]: any };
    [key: string]: any;
}

class BoundBlock<BLOCK = any> {
    block: BLOCK;
    name: String;
    element: HTMLElement;

    constructor(blockDef: BLOCK, prefix: String, element: HTMLElement) {
        this.block = blockDef;
        this.name = prefix;
        this.element = element;
    }

    getLabel(): string {
        throw new Error("Method not implemented.");
    }

    getState(): any {
        throw new Error("Method not implemented.");
    }

    getError(): any {
        throw new Error("Method not implemented.");
    }

    setState(state: any): void {
        throw new Error("Method not implemented.");
    }

    setError(errors: any): void {
        throw new Error("Method not implemented.");
    }
}

class Block {
    meta: BlockMeta;

    render(root: HTMLElement, id: string, ...args: any[]): BoundBlock {
        throw new Error("Method not implemented.");
    }
}

export {
    BoundBlock,
    Block,
    BlockMeta,
};