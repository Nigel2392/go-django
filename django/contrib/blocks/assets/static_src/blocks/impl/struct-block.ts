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
    wrapper: HTMLElement;
    subBlocks: { [key: string]: Block };

    constructor(meta: any, widget: HTMLElement, blockDef: BlockDef) {
        super(widget, blockDef);

        this.wrapper = widget;        
        
        this.subBlocks = {};
        for (const key in meta.childBlocks) {
            var childBlock = meta.childBlocks[key];
            this.subBlocks[key] = (childBlock as BlockDef).render();
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
    constructor(element: HTMLElement, config: Config) {
        super(element, config);
        console.log("StructBlockDef constructor", element, config);        
    }

    render(): any {
        return new StructBlock(
            this.config,
            this.element,
            this,
        );
    }

    config: Config;
}

export {
    StructBlock,
    StructBlockDef,
};