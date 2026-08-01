
import { Block, BlockMeta } from '../base';
import { jsx } from '../../../../../admin/static_src/jsx';
import tippy, { Instance as TippyInstance } from 'tippy.js';
import { BoundSequenceBlock, BoundSequenceBlockValue } from './sequence-block';

class BoundStreamBlock extends BoundSequenceBlock<StreamBlock> {
    dropdown: TippyInstance;

    _getChildBlock(value: any): Block | undefined {
        return this.block.childBlocks.get(value.type);
    }

    _renderBoundBlock(child: Block, placeholder: HTMLElement, id: string, name: String, value: any, error: any): BoundSequenceBlockValue {
        const boundValue: BoundSequenceBlockValue = {
            id: value.id,
            type: value.type,
            block: child.render(
                placeholder,
                id,
                name,
                value.data,
                error,
            )
        };

        return boundValue;
    }

    _defaultChildState(block: Block & { name: string }): any {
        const newValue: any = {
            id: '',
            type: block.name,
            data: null,
        };
        if (this.block.defaults[block.name]) {
            newValue.data = this.block.defaults[block.name];
        }
        return newValue;
    }

    _onAddClick(itemName: string | null, ev: MouseEvent) {
        ev?.preventDefault();

        if (this.block.childBlocks.size === 1) {
            const onlyType = Array.from(this.block.childBlocks.keys())[0];
            this._addBlock(itemName, this.block.childBlocks.get(onlyType), ev);
            return;
        }

        this.dropdown?.destroy();
        this.dropdown = tippy(ev.currentTarget as HTMLElement, {
            content: this._renderTypeMenu(itemName),
            allowHTML: true,
            interactive: true,
            trigger: 'manual',

            theme: 'dropdown',
            arrow: true,
            maxWidth: 350,
            placement: 'bottom',
        })

        this.dropdown.show();
    }

    _addBlock(itemName: string | null, block: Block, ev: MouseEvent): void {
        // Check if control key is pressed to allow multiple additions
        const isCtrlPressed = ev.ctrlKey || ev.metaKey;
        if (!isCtrlPressed) {
            this.dropdown?.destroy();
            this.dropdown = null;
        }

        super._addBlock(itemName, block, ev);
    }

    _getChildInputs(id: String, name: String, suffix: number, value: any): any {
        const typeKey = `${name}-${suffix}--type`;
        return (
            <input type="hidden" id={ typeKey } name={ typeKey } value={ value.type } />
        );
    }

    _renderTypeMenu(itemName: string | null) {
        return (
            <div class="dropdown">
                <div class="sequence-block-add-dropdown dropdown__content">
                    { Array.from(this.block.childBlocks.entries()).map(([type, block]: [string, Block]) => (
                        <button type="button" class="sequence-block-add-type-button" onClick={this._addBlock.bind(this, itemName, block)} data-block-type={type}>
                            { block.meta.label || type }
                        </button>
                    )) }
                </div>
            </div>
        )
    }
}

type childBlockMap = {
    name: string;
    block: Block;
}[];

class StreamBlock extends Block {
    name: string;
    defaults: { [key: string]: any };
    childBlocks: Map<string, Block>;

    constructor(name: string, childBlocks: childBlockMap, defaults: { [key: string]: any }, meta: BlockMeta) {
        super();
        this.name = name;
        this.meta = meta;
        this.childBlocks = new Map<string, Block>();
        this.defaults = defaults;
        for (let i = 0; i < childBlocks.length; i++) {
            this.childBlocks.set(childBlocks[i].name, childBlocks[i].block);
        }
    }

    render(root: HTMLElement, id: String, name: String, initialState: any, initialError: any): any {
        return new BoundStreamBlock(
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
    StreamBlock,
    BoundStreamBlock,
};
