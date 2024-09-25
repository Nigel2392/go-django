type Step = {
    start(...args: any): void;
    end(...args: any): void;
}

type GenericWorkflow = {
    steps: {
        [key: string]: Step;
    },
};

type Options = {
    steps: {
        [key: string]: Step;
    };
    currentStep?: string;
};

class Workflow implements GenericWorkflow {
    declare steps: {
        [key: string]: Step;
    };
    declare currentStep?: string;

    constructor(workflowConfig: Options) {
        this.steps = workflowConfig.steps || {};
        this.currentStep = null;
    }

    async exec(stepName: string, ...args: any) {
        await this.step(stepName, ...args)
    }

    async step(stepName: string, ...args: any) {
        const step = this.steps[stepName];
        if (this.currentStep) {
            this.steps[this.currentStep].end(...args);
        }
        
        if (step) {
            step.start(...args);
            this.currentStep = stepName;
        } else {
            console.error(`Step ${stepName} not found in workflow`);
            this.currentStep = null;
        }
    }
}
