import { BlockController } from "./blocks/controller";
import { Block } from "./blocks/base";

type BlockConstructor = new (...args: any) => Block;

class BlockApp {
    blocks: Record<string, BlockConstructor> = {};

    constructor(blocks?: Record<string, BlockConstructor>) {
        this.blocks = blocks || {};
        BlockController.classRegistry = this.blocks;
    }

    registerBlock(identifier: string, blockDefinition: BlockConstructor) {
        this.blocks[identifier] = blockDefinition
    }

    initBlock(...args: any): Block {
        const blockDef = this.blocks[args.blockType];
        return new blockDef(...args);
    }
}

export {
    BlockApp,
    BlockConstructor,
};
