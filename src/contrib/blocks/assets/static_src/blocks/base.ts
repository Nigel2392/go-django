

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

type Config = {
    id: string;
    type: string;
    name: string;
    label: string;
    html: string;
    errors: any;
    value: any;
    block: {
        element: {
            id: string;
            name: string;
            label: string;
            helpText: string;
            html: string;
        }
    };
    [key: string]: any;
}

class Block {
    blockDef: BlockDef;
    prefix: String;

    constructor(blockDef: BlockDef, prefix: String) {
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

class BlockDef {
    config: Config;
    element: HTMLElement;

    constructor(element: HTMLElement, config: Config) {
        this.element = element;
        this.config = config;
    }

    render(placeholder: HTMLElement, prefix: String, initialState: any, initialError: any): Block {
        throw new Error("Method not implemented.");
    }
}

export {
    Block,
    BlockDef,
    Config,
};