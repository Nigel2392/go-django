import { Application, Controller, ControllerConstructor } from "@hotwired/stimulus";


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

    private started: boolean = false;

    constructor(config: AppConfig) {
        this.controllers = config.controllers || {};
        this.stimulusApp = Application.start();
        window.Stimulus = this.stimulusApp;
        window.StimulusController = Controller;
    }

    registerController(identifier: string, controllerConstructor: ControllerConstructor) {
        if (this.started) {
            console.warn("AdminSite already started, registering new controllers to Stimulus application directly");
            this.stimulusApp.register(identifier, controllerConstructor);
            return;
        }
        this.controllers[identifier] = controllerConstructor;
    }

    start() {
        if (this.started) {
            console.warn("AdminSite already started");
            return;
        }
        
        this.started = true;

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