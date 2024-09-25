import { Application, ControllerConstructor, type Controller } from "@hotwired/stimulus";
import { BlockController } from "../blocks/controller";
import { BlockDef } from "../blocks/base";
import { Telepath } from "../telepath/telepath";

type BlockConstructor = new (...args: any) => BlockDef;

type App = {
    controllers: Record<string, ControllerConstructor>;
    registerBlock(identifier: string, blockDefinition: BlockConstructor): void;
    registerController(identifier: string, controllerConstructor: any): void;
    start(): void;
}

type AppConfig = {
    controllers?: Record<string, ControllerConstructor>;
    blocks?: Record<string, BlockConstructor>;
}

class DjangoApplication implements App {
    controllers: Record<string, ControllerConstructor> = {};
    blocks: Record<string, BlockConstructor> = {};
    stimulusApp: Application;
    telepath: Telepath;

    constructor(config: AppConfig) {
        this.controllers = config.controllers;
        this.blocks = config.blocks;
        this.stimulusApp = Application.start();
        window.Stimulus = this.stimulusApp;

        BlockController.classRegistry = this.blocks;
        this.telepath = new Telepath();
    }

    registerBlock(identifier: string, blockDefinition: BlockConstructor) {
        this.blocks[identifier] = blockDefinition
    }

    registerController(identifier: string, controllerConstructor: ControllerConstructor) {
        this.controllers[identifier] = controllerConstructor;
    }

    registerAdapter(name: string, adapter: any) {
        this.telepath.register(name, adapter);
    }

    unpack(data: any) {
        return this.telepath.unpack(data);
    }

    initBlock(...args: any): BlockDef {
        const blockDef = this.blocks[args.blockType];
        return new blockDef(...args);
    }

    start() {
        const keys = Object.keys(this.controllers);
        for (let i = 0; i < keys.length; i++) {
            const identifier = keys[i];
            const controllerConstructor = this.controllers[identifier];
            this.stimulusApp.register(identifier, controllerConstructor);
        }
    }
}

export {
    DjangoApplication,
    App,
    AppConfig,
    BlockConstructor,
    
};