import { BoundBlock, Block, Config } from '../base';
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

interface StructBlockMeta {
    childBlockDefs: BoundBlock[];
}

class BoundStructBlock extends BoundBlock {
    meta: StructBlockMeta;
    wrapper: HTMLElement;
    childBlocks: { [key: string]: BoundBlock };

    constructor(block: Block, placeHolder: HTMLElement, prefix: String, initialState: any, initialError: any) {
        super(block, prefix);

        this.childBlocks = {};

        for (let i = 0; i <block.config.childBlockDefs.length; i++) {
            const childBlock = block.config.childBlockDefs[i];
            const key = this.name + '-' + childBlock.name;

            
            //const childDom = (
            //    <div data-struct-field data-contentpath={ key }>
//
            //        @widgets.LabelComponent("struct-block", head.Value.Label(), id)
//
            //        {{ var newErrs = errors.Get(head.Key) }}
            //        <div data-struct-field-content>
            //            @renderForm(head.Value, id, key, valueMap[head.Key], newErrs, tplCtx)
            //        </div>
//
            //        @widgets.HelpTextComponent("struct-block", head.Value.HelpText())
            //    </div>
            //);

            //this.childBlocks[key] = (childBlock as Block).render(
            //    placeHolder,
            //    key,
            //    initialState[childBlock.name],
            //    initialError[childBlock.name],
            //);
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
            this.wrapper.style.backgroundColor = 'red';
        }
    }

}

class StructBlock extends Block {
    render(root: HTMLElement, name: String, initialState: any, initialError: any): any {
        return new BoundStructBlock(
            this,
            root,
            name,
            initialState,
            initialError,
        );
    }
}

export {
    StructBlock,
    BoundStructBlock,
};