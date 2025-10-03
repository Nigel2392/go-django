import { ClassCallController } from "../controllers"
import { Block, BlockDef, Config } from './base';



class BlockController extends ClassCallController<HTMLElement, Block> {
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

    initializeClass(klass: any): Block {

        if (this.element.classList.contains('block-initiated')) {
            return null
        }

        this.element.classList.add('block-initiated')

        const definition = window.telepath.unpack(this.classArgs)
        const block: BlockDef = new klass(this.element, definition)
        return block.render(
            this.element,
            block.config.block.element.name,
            block.config.value,
            block.config.errors,
        )
    }
}

export {
    BlockController,
};