import { BlockController } from "./blocks/controller";
import { BlockDef } from "./blocks/base";

type BlockConstructor = new (...args: any) => BlockDef;

class BlockApp {
    blocks: Record<string, BlockConstructor> = {};

    constructor(blocks?: Record<string, BlockConstructor>) {
        this.blocks = blocks || {};
        BlockController.classRegistry = this.blocks;
    }

    registerBlock(identifier: string, blockDefinition: BlockConstructor) {
        this.blocks[identifier] = blockDefinition
    }

    initBlock(...args: any): BlockDef {
        const blockDef = this.blocks[args.blockType];
        return new blockDef(...args);
    }
}

export {
    BlockApp,
    BlockConstructor,
};
