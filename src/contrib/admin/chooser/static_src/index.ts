import { Modal, ModalElement, ModalEvent } from "./modal/modal";
import { ChooserController, ChooserEvent } from "./controllers/chooser";

document.addEventListener('DOMContentLoaded', () => {
    window.Stimulus.register('chooser', ChooserController);
});
