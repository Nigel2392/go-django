import { ClassCallController } from "../controllers"
import { Block, BoundBlock } from './base';

type BlockElement = HTMLElement & {
    blocks?: {
        block: Block,
        bound: BoundBlock,
    }
}

class BlockController extends ClassCallController<BlockElement, BoundBlock> {
    declare idValue: string;
    declare argumentsValue: any[];

    static values = {
        ...ClassCallController.values,
        id: String,
        arguments: Array,
    }

    constructor(...args: ConstructorParameters<typeof ClassCallController<any, any>>) {
        super(...args);

        return new Proxy(this, {
            get: (target, prop: string | symbol, receiver) => {
                // normal properties/methods first
                const value = Reflect.get(target, prop, receiver)
                if (value !== undefined) return value
            
                // delegate block* methods
                if (typeof prop === "string" && prop.startsWith("block")) {
                    // "blockAdd" -> "add"
                    const raw = prop.slice(5)
                    const name = raw.charAt(0).toLowerCase() + raw.slice(1)
                    return (...args: any[]) => {
                        const el = target.element as BlockElement
                        const blocks = el.blocks
                        const fn = (blocks.bound as any)[name]
                        if (typeof fn !== "function") {
                            console.warn(`[BlockController] No block method '${name}' found on`, blocks.bound)
                            return
                        }
                        return fn.apply(blocks.bound, args)
                    }
                }
            
                return value
            },
        })
    }

    initializeClass(klass: any): BoundBlock {

        if (this.element.classList.contains('block-initiated')) {
            return null
        }

        this.element.classList.add('block-initiated')

        const definition: Block = window.telepath.unpack(this.classArgs)
        const bound = definition.render(
            this.element, this.idValue, ...this.argumentsValue,
        )
        this.element.blocks = {
            block: definition,
            bound: bound,
        }
        return bound;
    }
}

export {
    BlockController,
};