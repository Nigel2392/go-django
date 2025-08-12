import { Controller, ActionEvent } from "@hotwired/stimulus";
import { Modal } from "../modal/modal";

type ChooserEvent = Event & { detail: { action: "open" | "close", modal: ChooserController }};

type ChooserControllerElement = HTMLElement & { chooserController?: ChooserController, dataset: { chooserController: string } };

type ChooserResponse = {
    html:         string;
    preview?:     string;
    pk?:          string;
    errors?:      string[];
}

function newChooserEvent(action: "open" | "close", modal: ChooserController, event?: Event): ChooserEvent {
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
    modalWrapper: HTMLElement;

    static targets = ["open", "clear", "preview", "input", "link"];
    static values = {
        listurl:   { type: String },
        createurl: { type: String },
        updateurl: { type: String },
        title:     { type: String },
    };

    declare readonly listurlValue:   string;
    declare readonly createurlValue: string;
    declare readonly titleValue:     string;

    declare readonly openTarget:    HTMLButtonElement;
    declare readonly clearTarget:   HTMLButtonElement;
    declare readonly previewTarget: HTMLDivElement;
    declare readonly inputTarget:   HTMLInputElement;
    declare readonly linkTargets:   HTMLAnchorElement[];

    connect() {
        this.modalWrapper = document.getElementById("godjango-modal-wrapper");
        if (!this.modalWrapper) {
            this.modalWrapper = document.createElement("div");
            this.modalWrapper.id = "godjango-modal-wrapper";
            this.modalWrapper.className = "godjango-modal-wrapper";
            document.body.appendChild(this.modalWrapper);
        }

        this.element.chooserController = this;
    }

    disconnect() {
        this.modal = null;
        this.element.chooserController = null;
        this.element.dataset.chooserController = "false";
        this.modal.disconnect();
    }

    async fetch(url: string, method: string = "GET", body?: any, headers?: HeadersInit): Promise<ChooserResponse> {
        const opts: RequestInit = {
            method: method,
            headers: {
                "Accept": "application/json",
                ...headers,
            },
        }

        if (body && opts.method.toUpperCase() !== "GET") {
            opts.body = body instanceof FormData ? body : JSON.stringify(body);
        }
        
        const request = new Request(url, opts);
        const response = await fetch(request);
        if (!response.ok) {
            throw new Error(`Failed to fetch ${url}: ${response.statusText}`);
        }

        var data: ChooserResponse = await response.json();
        if (!data.html && (!data.pk && !data.preview)) {
            throw new Error(`Invalid response from ${url}: missing html in ${Object.keys(data)}`);
        }

        if (data.errors && data.errors.length > 0) {
            for (const error of data.errors) {
                console.error(`Error from ${url}:`, error);
            }
        }

        return data;
    }

    private get searchForm(): HTMLFormElement | null {
        return this.modal.content.querySelector(".godjango-chooser-list-form");
    }

    async loadModalContent(url: string, method: string = "GET", body?: any, headers?: HeadersInit): Promise<ChooserResponse> {

        console.debug("Loading modal content from:", url);

        try {
            const data = await this.fetch(url, method, body, headers);

            if (data.pk) {
                return data;
            }

            this.modal.content = data.html;
        } catch (error) {
            console.error("Error loading modal content:", error);
        }

        return null;
    }

    select(value: string, previewText: string) {
        this.inputTarget.value = value;
        this.previewTarget.innerHTML = previewText;
    }

    async setup() {
        this.modal.title = `<h1>${this.titleValue}</h1>`;
        await this.showList();
    }

    async teardown() {
        this.modal.disconnect();
    }

    async open(event?: ActionEvent) {
        this.modal = new Modal(this.modalWrapper, {
            opened: true,
            executeScriptsOnSet: true,
            onClose: async (event) => {
                await this.teardown();
                await this.element.dispatchEvent(newChooserEvent("close", this, event));
            },
        });
        await this.setup();
        await this.element.dispatchEvent(newChooserEvent("open", this, event));
    }

    async close(event?: ActionEvent) {
        await this.teardown();
        await this.element.dispatchEvent(newChooserEvent("close", this, event));
    }

    async clear(event?: ActionEvent) {
        if (this.inputTarget.value === "") {
            return;
        }
        
        this.inputTarget.value = "";
        this.previewTarget.innerHTML = "";
        
        const currentColor = getComputedStyle(this.element).borderColor;
        this.element.animate([{ borderColor: "red" }, { borderColor: currentColor }], {
            fill: "forwards",
            duration: 300,
            easing: "ease-out",
        });
    }

    async showList(url: string = this.listurlValue) {
        await this.loadModalContent(url);

        var rows = this.modal.content.querySelectorAll(".godjango-chooser-list-group") as NodeListOf<HTMLElement>;
        rows.forEach((row) => {
            row.addEventListener("click", () => {
                var value = row.dataset.chooserValue;
                var previewText = row.dataset.chooserPreview;
                this.select(value, previewText);
                this.close();
            });
        });

        this.searchForm.addEventListener("submit", async (event) => {
            event.preventDefault();

            const formData = new FormData(this.searchForm);
            const searchParams = new URLSearchParams(formData as any);
            const url = `${this.listurlValue}?${searchParams.toString()}`;
            await this.showList(url);
        });


        var links = this.modal.content.querySelectorAll(".pagination a") as NodeListOf<HTMLAnchorElement>;
        links.forEach((link) => {
            link.addEventListener("click", async (event) => {
                event.preventDefault();

                const href = link.getAttribute("href");
                if (href) {
                    await this.showList(href);
                } else {
                    console.warn("Chooser link has no href:", link);
                }
            });
        });

        var createNewButton = this.modal.content.querySelector(".godjango-chooser-list-create") as HTMLButtonElement;
        createNewButton.addEventListener("click", async (event) => {
            event.preventDefault();
            await this.showCreate();
        });
    }

    async showCreate(url: string = this.createurlValue, method: string = "GET", body: any = null) {
        let data = await this.loadModalContent(url, method, body);
        if (data) {
            this.select(data.pk, data.preview);
            await this.close();
            return;
        }

        const backToList = this.modal.content.querySelector("#back-to-list") as HTMLButtonElement;
        backToList.addEventListener("click", async (event) => {
            event.preventDefault();
            await this.showList();
        });

        const form = this.modal.content.querySelector("form") as HTMLFormElement;
        form.addEventListener("submit", async (event) => {
            event.preventDefault();

            const formData = new FormData(form);
            await this.showCreate(url, "POST", formData);
        });
    }
}

export {
    ChooserEvent,
    ChooserController,
};
