import { ClassCallController } from "../controllers"
import { Block, BoundBlock, Config } from './base';

type BlockElement = HTMLElement & {
    blocks?: {
        block: Block,
        bound: BoundBlock,
    }
}

class BlockController extends ClassCallController<BlockElement, BoundBlock> {
    declare classErrorsValue: string
    declare hasClassErrorsValue: boolean
    declare _classErrors: any[]

    static values = {
        ...ClassCallController.values,
        classErrors: String,
    }

    get classErrors() {
        if (!this.hasClassErrorsValue) {
            return []
        }
        if (!this._classErrors) {
            this._classErrors = JSON.parse(this.classErrorsValue)
        }
        return this._classErrors
    }

    initializeClass(klass: any): BoundBlock {

        if (this.element.classList.contains('block-initiated')) {
            return null
        }

        this.element.classList.add('block-initiated');

        const definition = window.telepath.unpack(this.classArgs);
        const block: Block = new klass(definition);
        const bound = block.render(
            this.element,
            block.config.name,
            block.config.value,
            block.config.errors,
        )
        this.element.blocks = {
            block: block,
            bound: bound,
        }
        return bound;
    }
}

export {
    BlockController,
};