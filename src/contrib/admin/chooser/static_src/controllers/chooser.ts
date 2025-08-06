import { Controller, ActionEvent } from "@hotwired/stimulus";
import { Modal } from "../modal/modal";

type ChooserEvent = Event & { detail: { action: "open" | "close", modal: ChooserController }};

type ChooserControllerElement = HTMLElement & { modalController?: ChooserController, dataset: { modalController: string } };

function newChooserEvent(action: "open" | "close", modal: ChooserController, event?: ActionEvent): ChooserEvent {
    return new CustomEvent("modal:" + action, {
        detail: {
            action: action,
            modal: modal,
            originalEvent: event,
        }
    }) as ChooserEvent;
}

class ChooserController extends Controller<any> {
    modal: Modal;

    connect() {
        this.tryClose();
        this.element.modalController = this;
        this.modal = new Modal(this.element);
    }

    disconnect() {
        this.modal = null;
        this.element.modalController = null;
        this.element.dataset.modalController = "false";
        this.element.remove();
    }

    tryClose() {
        var modals = document.querySelectorAll("[data-controller='modal']");
        if (modals.length == 1) {
            return;
        }

        console.warn(
            "Multiple modal controllers found on the page. This may cause unexpected behavior.",
        );

        modals.forEach((modal: ChooserControllerElement) => {
            if (modal !== this.element) {
                modal.modalController?.disconnect();
            }
        });
    }

    open(event?: ActionEvent) {
        this.modal.open(event);
        this.element.dispatchEvent(newChooserEvent("open", this, event));
    }

    close(event?: ActionEvent) {
        this.modal.close(event);
        this.element.dispatchEvent(newChooserEvent("close", this, event));
    }
}

export {
    ChooserEvent,
    ChooserController,
};
