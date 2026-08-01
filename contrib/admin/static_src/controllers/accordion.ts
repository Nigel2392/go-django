import { Controller, ActionEvent } from "@hotwired/stimulus";

class AccordionController extends Controller<any> {
    static values = {
        cookieName: String
    }

    declare cookieNameValue: string;
    declare hasCookieNameValue: boolean;

    connect() {
        this.element.accordionController = this;
        this.element.dataset.accordionController = "true";

        if (this.hasCookieNameValue && window.getCookie(this.cookieNameValue) === "open") {
            this.open();
        }
    }

    toggle(event?: ActionEvent) {
        if (this.element.classList.contains("open")) {
            this.close(event);
        } else {
            this.open(event);
        }
    }

    open(event?: ActionEvent) {
        if (!this.element.classList.contains("open")) {
            this.element.classList.add("open");
            this.element.setAttribute("aria-expanded", "true");
        }
    }

    close(event?: ActionEvent) {
        if (this.element.classList.contains("open")) {
            this.element.classList.remove("open");
            this.element.setAttribute("aria-expanded", "false");
        }
    }
}


export {
    AccordionController,
};
