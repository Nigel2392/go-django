import { Block, BlockDef, Config } from '../base';

class FieldBlock extends Block {
    input: HTMLInputElement;

    constructor(id: string, name: string, widget: HTMLElement, blockDef: BlockDef) {
        super(widget, blockDef);

        this.input = document.getElementById(id) as HTMLInputElement;
    }

    getLabel(): string {
        return this.input.value;
    }

    getState(): any {
        return this.input.value;
    }

    setState(state: any): void {
        this.input.value = state;
    }

    setError(errors: string[]): void {
        this.input.style.backgroundColor = 'red';
    }

}

class FieldBlockDef extends BlockDef {
    constructor(config: Config) {
        super(config);
        this.config = config;
    }

    render(): any {
        return new FieldBlock(
            this.config.id,
            this.config.name,
            document.getElementById(
                this.config.id,
            ),
            this,
        );
    }

    config: Config;
}

export {
    FieldBlock,
    FieldBlockDef,
};