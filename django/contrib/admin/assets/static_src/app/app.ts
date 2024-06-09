import { Application, ControllerConstructor, type Controller } from "@hotwired/stimulus";


type App = {
    controllers: Record<string, ControllerConstructor>;
    registerController(identifier: string, controllerConstructor: any): void;
    start(): void;
}

type AppConfig = {
    controllers?: Record<string, ControllerConstructor>;
}

class AdminSite implements App {
    controllers: Record<string, ControllerConstructor> = {};
    stimulusApp: Application;

    constructor(config: AppConfig) {
        this.controllers = config.controllers || {};
        this.stimulusApp = Application.start();
        window.Stimulus = this.stimulusApp;
    }

    registerController(identifier: string, controllerConstructor: ControllerConstructor) {
        this.controllers[identifier] = controllerConstructor;
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
    AdminSite,
    App,
    AppConfig,
};