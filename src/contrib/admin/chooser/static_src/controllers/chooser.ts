import { Controller, ActionEvent } from "@hotwired/stimulus";
import { Modal } from "../modal/modal";

type ChooserEvent = Event & { detail: { action: "open" | "close", modal: ChooserController }};

type ChooserControllerElement = HTMLElement & { chooserController?: ChooserController, dataset: { chooserController: string } };

type ChooserResponse = {
    html:         string;
    preview_html: string;
    errors?:      string[];
}

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

    static targets = ["open", "clear", "preview", "input", "modal"];
    static values = {
        listurl:   { type: String },
        createurl: { type: String },
        updateurl: { type: String },
        title:     { type: String },
    };

    declare readonly listurlValue:   string;
    declare readonly createurlValue: string;
    declare readonly updateurlValue: string;
    declare readonly titleValue:     string;

    declare readonly openTarget:    HTMLButtonElement;
    declare readonly clearTarget:   HTMLButtonElement;
    declare readonly previewTarget: HTMLDivElement;
    declare readonly inputTarget:   HTMLInputElement;
    declare readonly modalTarget:   HTMLDivElement;

    connect() {
        this.tryClose();
        this.element.chooserController = this;
        this.modal = new Modal(this.modalTarget);
        this.modal.title = `<h1>${this.titleValue}</h1>`;
    }

    disconnect() {
        this.modal = null;
        this.element.chooserController = null;
        this.element.dataset.chooserController = "false";
        this.element.remove();
    }

    tryClose() {
        var choosers = document.querySelectorAll("[data-controller='chooser']");
        if (choosers.length == 1) {
            return;
        }

        console.warn(
            "Multiple chooser controllers found on the page. This may cause unexpected behavior.",
        );

        choosers.forEach((chooser: ChooserControllerElement) => {
            if (chooser !== this.element) {
                chooser.chooserController?.disconnect();
            }
        });
    }

    async fetch(url: string, method: string = "GET", body?: any, headers?: HeadersInit): Promise<ChooserResponse> {
        const response = await fetch(url, {
            method: method,
            headers: {
                "Content-Type": "application/json",
                ...headers,
            },
            body: body ? JSON.stringify(body) : undefined,
        });

        if (!response.ok) {
            throw new Error(`Failed to fetch ${url}: ${response.statusText}`);
        }

        var data: ChooserResponse = await response.json();
        if (!data.html || !data.preview_html) {
            throw new Error(`Invalid response from ${url}: missing html or preview_html`);
        }

        if (data.errors && data.errors.length > 0) {
            for (const error of data.errors) {
                console.error(`Error from ${url}:`, error);
            }
        }

        return data;
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
