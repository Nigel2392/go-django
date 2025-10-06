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

    constructor(blockDef: Block, placeHolder: HTMLElement, prefix: String, initialState: any, initialError: any) {
        super(blockDef, prefix);

        this.childBlocks = {};

        for (let i = 0; i <blockDef.config.childBlockDefs.length; i++) {
            const childBlock = blockDef.config.childBlockDefs[i];
            const key = this.prefix + '-' + childBlock.name;

            console.log("StructBlock constructor", childBlock, key);
            
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
        for (let i = 0; i < this.blockDef.config.childBlocks.length; i++) {
            const block = this.blockDef.config.childBlocks[i];
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
    render(placeholder: HTMLElement, prefix: String, initialState: any, initialError: any): any {
        console.log("StructBlockDef render 1", placeholder);
        console.log("StructBlockDef render 2", prefix);
        console.log("StructBlockDef render 3", initialState);
        console.log("StructBlockDef render 4", initialError);
        return new BoundStructBlock(
            this,
            placeholder,
            prefix,
            initialState,
            initialError,
        );
    }
}

export {
    StructBlock,
    BoundStructBlock,
};