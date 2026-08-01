type ClassArgs = {
    [key: string]: any
}

class ClassCallController<ET extends Element, CT extends any> extends window.StimulusController<ET> {
    declare classArgsValue: string
    declare hasClassArgsValue: boolean
    declare classPathValue: string
    declare _classArgs: ClassArgs

    static classRegistry: Record<string, any> = {}
    static values = {
        classPath: String,
        classArgs: String,
    }

    static registerClass(klass: any, path: string) {
        this.classRegistry[path] = klass
    }

    get classArgs(): ClassArgs {
        if (!this.hasClassArgsValue) {
            return {}
        }
        if (!this._classArgs) {
            this._classArgs = JSON.parse(this.classArgsValue)
        }
        return this._classArgs
    }

    connect() {
        const constructorClass = this.constructor as typeof ClassCallController
        if (this.classPathValue in constructorClass.classRegistry) {
            const klass = constructorClass.classRegistry[this.classPathValue]
            const klassInstance = this.initializeClass(klass)
            const anyElem = this.element as any

            anyElem[`${this.identifier}Class`] = klassInstance
            anyElem[`${this.identifier}Controller`] = this
        } else {
            console.error(`Class ${this.classPathValue} not found in registered classes`)
        }
    }

    initializeClass(klass: any): CT {
        return new klass(this.classArgs)
    }
}

export {
    ClassCallController,
};