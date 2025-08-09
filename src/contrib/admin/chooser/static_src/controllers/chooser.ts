import { Controller, ActionEvent } from "@hotwired/stimulus";
import { Modal } from "../modal/modal";

type ChooserEvent = Event & { detail: { action: "open" | "close", modal: ChooserController }};

type ChooserControllerElement = HTMLElement & { chooserController?: ChooserController, dataset: { chooserController: string } };

type ChooserResponse = {
    html:         string;
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
        if (!data.html) {
            throw new Error(`Invalid response from ${url}: missing html in ${Object.keys(data)}`);
        }

        if (data.errors && data.errors.length > 0) {
            for (const error of data.errors) {
                console.error(`Error from ${url}:`, error);
            }
        }

        return data;
    }

    async loadModalContent(url: string) {

        console.debug("Loading modal content from:", url);

        try {
            const data = await this.fetch(url);
            this.modal.content = data.html;
        } catch (error) {
            console.error("Error loading modal content:", error);
        }

        var groups = this.modal.content.querySelectorAll(".godjango-chooser-list-group") as NodeListOf<HTMLElement>;
        groups.forEach((group) => {
            group.addEventListener("click", () => {
                console.log("Group clicked:", Object.keys(group.dataset), group.dataset);
                var value = group.dataset.chooserValue;
                var previewText = group.dataset.chooserPreview;
                this.select(value, previewText);
                this.close();
            });
        });
    }

    select(value: string, previewText: string) {
        this.inputTarget.value = value;
        this.previewTarget.innerHTML = previewText;
    }

    async setup() {
        this.modal.title = `<h1>${this.titleValue}</h1>`;
        await this.loadModalContent(this.listurlValue);
    }

    async teardown() {

    }

    async open(event?: ActionEvent) {
        this.modal.open(event);
        await this.setup();
        await this.element.dispatchEvent(newChooserEvent("open", this, event));
    }

    async close(event?: ActionEvent) {
        this.modal.close(event);
        await this.teardown();
        await this.element.dispatchEvent(newChooserEvent("close", this, event));
    }
}

export {
    ChooserEvent,
    ChooserController,
};
