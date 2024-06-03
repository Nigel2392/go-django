import { ClassCallController } from "../controllers"


class BlockController extends ClassCallController<HTMLElement, BlockDef> {
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

    initializeClass(klass: any): BlockDef {

        console.log('initializeClass', klass)
        console.log('classArgs', this.classArgs)
        console.log('classErrors', this.classErrors)

        const block: BlockDef = new klass(this.classArgs, this.classErrors)

        if (this.classErrors.length > 0) {
            this.element.style.backgroundColor = 'red'
            block.setError(this.classErrors)
        }

        return block
    }
}

export {
    BlockController,
};