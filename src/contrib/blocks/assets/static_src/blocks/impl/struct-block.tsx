import { BoundBlock, Block, BlockMeta } from '../base';
import { jsx } from '../../../../../editor/features/links/static_src/jsx';

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

type childBlockMap = {
    name: string;
    block: Block;
}[];

class BoundStructBlock extends BoundBlock {
    childBlocks: { [key: string]: BoundBlock };

    constructor(block: StructBlock, placeHolder: HTMLElement, name: String, id: String, initialState: any, initialError: any) {
        super(block, name, placeHolder);

        this.element.dataset.structBlock = 'true';

        this.childBlocks = {};

        initialState = initialState || {};
        initialError = initialError || {};

        var keys = Object.keys(block.children);
        for (let i = 0; i < keys.length; i++) {
            const childBlockName = keys[i];
            const childBlock = block.children[childBlockName];
            const key = this.name + '-' + childBlockName;
            const idKey = id + '-' + childBlockName;
            const childDom = (
                <div data-struct-field data-contentpath={ key }>
                    <div data-struct-field-content>
                    </div>
                </div>
            );

            placeHolder.appendChild(childDom);

            this.childBlocks[key] = (childBlock as Block).render(
                childDom.firstElementChild as HTMLElement,
                idKey,
                key,
                initialState[childBlockName],
                initialError[childBlockName],
            );
        }
    }

    getLabel(): string {
        for (let i = 0; i < this.block.config.childBlocks.length; i++) {
            const block = this.block.config.childBlocks[i];
            let label = block.getLabel();
            if (label) {
                return label;
            }
        }
        return '';
    }

    getState(): any {
        const state: any = {};
        for (const key in this.childBlocks) {
            state[key] = this.childBlocks[key].getState();
        }
        return state;
    }

    setState(state: any): void {
        for (const key in this.childBlocks) {
            this.childBlocks[key].setState(state[key]);
        }
    }

    setError(errors: any): void {
        if (!errors) {
            return;
        }

        if (errors.blockErrors) {
            for (const key in errors.blockErrors) {
                this.childBlocks[key].setError(errors.blockErrors[key]);
            }
        }

        if (errors.nonFieldErrors) {
            this.element.style.backgroundColor = 'red';
        }
    }

}

class StructBlock extends Block {
    name: string;
    children: { [key: string]: Block };

    constructor(name: string, children: childBlockMap, meta: BlockMeta) {
        super();
        this.name = name;
        this.meta = meta;
        this.children = {};
        for (let i = 0; i < children.length; i++) {
            this.children[children[i].name] = children[i].block;
        }
    }

    render(root: HTMLElement, id: string, name: string, initialState: any, initialError: any): any {
        return new BoundStructBlock(
            this,
            root,
            name,
            id,
            initialState,
            initialError,
        );
    }
}

export {
    StructBlock,
    BoundStructBlock,
};