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


// Actual bound configuration passed to BlockDef
type Config<T = any> = {
    id: string;
    type?: string;
    name: string;
    label?: string;
    errors?: any;
    value?: any;
    attrs?: { [key: string]: any };
    helpText?: string;
    block: T;
    [key: string]: any;
}

class BoundBlock<BLOCK = any> {
    block: BLOCK;
    name: String;

    constructor(blockDef: BLOCK, prefix: String) {
        this.block = blockDef;
        this.name = prefix;
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

class Block<BLOCK = any, T1 = Config<BLOCK>> {
    config: T1;

    constructor(config: T1) {
        this.config = config;
    }

    render(root: HTMLElement, name: String, initialState: any, initialError: any): BoundBlock {
        throw new Error("Method not implemented.");
    }
}

export {
    BoundBlock,
    Block,
    Config,
};