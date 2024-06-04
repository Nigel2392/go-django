import { Block, BlockDef, Config } from '../base';

const getElementIfAttr = (parent: HTMLElement, attr: string, value?: string): HTMLElement => {
    for (var i = 0; i < parent.children.length; i++) {
        if (parent.children[i].hasAttribute(attr)) {

            if (value && parent.children[i].getAttribute(attr) !== value) {
                continue;
            }

            return parent.children[i] as HTMLElement;
        }
    }
    return null;
};

class StructBlock extends Block {
    wrapper: HTMLDivElement;
    subBlocks: { [key: string]: Block };

    constructor(id: string, name: string, widget: HTMLElement, blockDef: BlockDef) {
        super(widget, blockDef);

        this.wrapper = document.getElementById(id) as HTMLDivElement;
        this.subBlocks = {};
        for (let i = 0; i < this.wrapper.children.length; i++) {
            const blockElem = this.wrapper.children[i] as HTMLElement;
            const blockFieldContent = getElementIfAttr(blockElem, "data-struct-field-content");
            if (!blockFieldContent) {
                console.log("no blockFieldContent found for data-struct-field-content", blockElem);
                continue;
            }
            const block = getElementIfAttr(blockFieldContent, "data-controller", "block");
            if (!block) {
                console.log("no block found for data-controller=block", blockElem);
                continue;
            }
            console.log("block", block, (block as any).blockClass);
            block.addEventListener("blocks:init", (event: any) => {
                console.log("blocks:init", event);
            });
        }
    }

    getLabel(): string {
        for (const key in this.subBlocks) {
            const block = this.subBlocks[key];
            if (block.getLabel()) {
                return block.getLabel();
            }
        }
        return '';
    }

    getState(): any {
        const state: any = {};
        for (const key in this.subBlocks) {
            state[key] = this.subBlocks[key].getState();
        }
        return state;
    }

    setState(state: any): void {
        for (const key in this.subBlocks) {
            this.subBlocks[key].setState(state[key]);
        }
    }

    setError(errors: any): void {
        if (!errors) {
            return;
        }

        if (errors.blockErrors) {
            for (const key in errors.blockErrors) {
                this.subBlocks[key].setError(errors.blockErrors[key]);
            }
        }

        if (errors.nonFieldErrors) {
            this.wrapper.style.backgroundColor = 'red';
        }
    }

}

class StructBlockDef extends BlockDef {
    constructor(config: Config) {
        super(config);
        this.config = config;
    }

    render(): any {
        console.log("rendering StructBlock", this.config.id);
        return new StructBlock(
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
    StructBlock,
    StructBlockDef,
};