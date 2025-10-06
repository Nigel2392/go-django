

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

// 
type ConfigElement = {
    id: string;
    name: string;
    label: string;
    helpText: string;
    required: boolean;
    html: string;
}

// Static- ish configuration passed to the frontend
// 
// This is based on the block definition, without any of the 
// dynamic values passed, such as id, name, value, errors, etc.
type ConfigBlock = {
    element: ConfigElement;
}

// Actual bound configuration passed to BlockDef
type Config = {
    id: string;
    type?: string;
    name: string;
    label?: string;
    html?: string;
    errors?: any;
    value?: any;
    block: ConfigBlock;
    [key: string]: any;
}

class BoundBlock {
    blockDef: Block;
    prefix: String;

    constructor(blockDef: Block, prefix: String) {
        this.blockDef = blockDef;
        this.prefix = prefix;
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

class Block<T = Config> {
    config: T;
    element: HTMLElement;

    constructor(element: HTMLElement, config: T) {
        this.element = element;
        this.config = config;
    }

    render(placeholder: HTMLElement, prefix: String, initialState: any, initialError: any): BoundBlock {
        throw new Error("Method not implemented.");
    }
}

export {
    BoundBlock,
    Block,
    Config,
    ConfigElement,
    ConfigBlock,
};