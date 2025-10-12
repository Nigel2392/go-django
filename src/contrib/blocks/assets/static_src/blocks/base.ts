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

type BlockError = {
    [key: string]: BlockError | string[];
    errors?: BlockError | string[];
    nonBlockErrors?: BlockError | string[];
};

type BlockMeta = {
    label?: string;
    helpText?: string;
    required?: boolean;
    default?: any;
    attrs?: { [key: string]: any };
    [key: string]: any;
}

class BoundBlock<BLOCK = any, ELEM = HTMLElement> {
    block: BLOCK;
    name: String;
    element: ELEM;

    constructor(blockDef: BLOCK, prefix: String, element: ELEM) {
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

    setError(errors: BlockError | string[]): void {
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