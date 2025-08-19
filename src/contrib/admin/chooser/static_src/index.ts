import { Modal, ModalElement, ModalEvent } from "./modal/modal";
import { ChooserController, ChooserEvent } from "./controllers/chooser";
import { Chooser } from "./chooser/chooser";

document.addEventListener('DOMContentLoaded', () => {
    window.Stimulus.register('chooser', ChooserController);
});

(window as any).Chooser = Chooser;
