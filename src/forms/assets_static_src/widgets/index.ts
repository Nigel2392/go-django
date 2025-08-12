import { MultipleSelectController } from "./multiple-select/multiple_select";

const initialize = () => {
    console.debug('MultipleSelectController initialized');
    (window as any).Stimulus.register('multiple-select', MultipleSelectController);
}

if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initialize);
} else {
    initialize();
}
