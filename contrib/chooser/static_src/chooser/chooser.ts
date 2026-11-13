import { Modal } from "./modal";

type ChooserResponse = {
    html:         string;
    preview?:     string;
    pk?:          string;
    data?:        any;
    errors?:      string[];
}

type ChooserConfig = {
    title:      string;
    listurl:    string;
    modalClass?: string;
    onChosen?:   (value?: string, previewText?: string, data?: any) => void;
    onOpen?:     () => void;
    onClosed?:   () => void;
};

class Chooser {
    modal: Modal;
    modalWrapper: HTMLElement;
    config: ChooserConfig;


    constructor(config: ChooserConfig) {
        this.config = config;

        if (!this.config.modalClass) {
            this.config.modalClass = "godjango-modal-wrapper";
        }

        this.modalWrapper = document.getElementById(this.config.modalClass);
        if (!this.modalWrapper) {
            this.modalWrapper = document.createElement("div");
            this.modalWrapper.id = this.config.modalClass;
            this.modalWrapper.className = this.config.modalClass;
            document.body.appendChild(this.modalWrapper);
        }
    }

    disconnect() {
        this.modal = null;
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

        if (body && opts.method?.toUpperCase() !== "GET") {
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

    private get searchForm(): HTMLFormElement | undefined {
        return this.modal.content?.querySelector(".godjango-chooser-list-form");
    }

    async loadModalContent(url: string, method: string = "GET", body?: any, headers?: HeadersInit): Promise<ChooserResponse | undefined> {

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

        return undefined;
    }

    select(value?: string, previewText?: string, data?: any) {
        if (!this.config.onChosen) {
            console.error("Chooser onChosen callback is not defined");
            return;
        }
        this.config.onChosen(value, previewText, data);
    }

    async setup() {
        this.modal.title = `<h1>${this.config.title}</h1>`;
        await this.showList();
    }

    async teardown() {
        this.modal.disconnect();
    }

    async open() {
        this.modal = new Modal(this.modalWrapper, {
            opened: true,
            executeScriptsOnSet: true,
            onOpen: () => {
                if (this.config.onOpen) {
                    this.config.onOpen();
                }
            },
            onClose: async (event: any) => {
                await this.teardown();

                if (this.config.onClosed) {
                    this.config.onClosed();
                }
            },
        });
        await this.setup();
    }

    async close() {
        this.modal.close();
        await this.teardown();
    }

    async showList(url: string = this.config.listurl) {
        await this.loadModalContent(url);

        var rows = this.modal.content?.querySelectorAll(".godjango-chooser-list-group") as NodeListOf<HTMLElement>;
        rows.forEach((row) => {
            row.addEventListener("click", (e: Event) => {

                if ((e.target as HTMLElement).classList.contains("chooser-list-link") || (e.target as HTMLElement).closest(".chooser-list-link")) {
                    console.debug("Clicked on link inside chooser row, ignoring row click");
                    return;
                }
                
                e.stopPropagation();
                e.preventDefault();
                var value = row.dataset.chooserValue;
                var previewText = row.dataset.chooserPreview;
                this.select(value, previewText, JSON.parse(row.dataset.chooserData || "{}"));
                this.close();
            });
        });

        if (this.searchForm) {
            this.searchForm.addEventListener("submit", async (event) => {
                event.preventDefault();

                const formData = new FormData(this.searchForm);
                const searchParams = new URLSearchParams(formData as any);
                const url = `${this.config.listurl}${this.config.listurl.includes("?") ? "&" : "?"}${searchParams.toString()}`;
                await this.showList(url);
            });
        }

        var links = this.modal.content?.querySelectorAll(".pagination a, .chooser-list-link") as NodeListOf<HTMLAnchorElement>;
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
    }
}

export {
    Chooser,
};
