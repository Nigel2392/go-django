

type Block = {
    constructor(...args: any): void;
    widget: HTMLElement;
    blockDef: BlockDef;

    getLabel(): string;
    getState(): any;
    getError(): any;
    setState(state: any): void;
    setError(errors: any): void;
}
