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

        const block: BlockDef = new klass(this.classArgs, this.classErrors)

        if (this.classErrors && this.classErrors.length > 0) {
            this.element.style.backgroundColor = 'red'
        }

        return block.render()
    }
}

export {
    BlockController,
};