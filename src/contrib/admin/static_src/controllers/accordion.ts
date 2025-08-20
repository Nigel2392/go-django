import { Controller, ActionEvent } from "@hotwired/stimulus";

class AccordionController extends Controller<any> {
    declare contentTarget: HTMLElement;

    connect() {
        this.element.accordionController = this;
        this.element.dataset.accordionController = "true";
    }

    toggle(event?: ActionEvent) {
        if (this.element.classList.contains("open")) {
            this.close(event);
        } else {
            this.open(event);
        }
    }

    private open(event?: ActionEvent) {
        this.element.classList.add("open");
        this.element.setAttribute("aria-expanded", "true");
    }

    private close(event?: ActionEvent) {
        this.element.classList.remove("open");
        this.element.setAttribute("aria-expanded", "false");
    }
}


export {
    AccordionController,
};
