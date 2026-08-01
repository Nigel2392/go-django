import { jsx } from "../../../static_src/jsx";

type ModalEvent = Event & { detail: { action: "modal:open" | "modal:close", modal: Modal }};
type ModalElement = HTMLElement & { modal?: Modal, dataset: { modal: string } };
type ActionMap = {
    [action: string]: (event?: MouseEvent | KeyboardEvent) => void;
};

class Elements {
    root: ModalElement;
    modal: HTMLElement;

    private _dialog: HTMLDialogElement | null = null;
    private _controls: HTMLElement | null = null;
    private _title: HTMLElement | null = null;
    private _content: HTMLElement | null = null;
    private _footer: HTMLElement | null = null;

    constructor(root: ModalElement) {
        this.root = root;
    }

    get dialog(): HTMLDialogElement {
        if (!this._dialog) {
            this._dialog = this.root.querySelector("dialog");
        }
        return this._dialog;
    }

    get controls(): HTMLElement {
        if (!this._controls) {
            this._controls = this.root.querySelector(".godjango-modal-controls");
        }
        return this._controls;
    }

    get title(): HTMLElement {
        if (!this._title) {
            this._title = this.root.querySelector(".godjango-modal-header");
        }
        return this._title;
    }

    get content(): HTMLElement {
        if (!this._content) {
            this._content = this.root.querySelector(".godjango-modal-content");
        }
        return this._content;
    }

    get footer(): HTMLElement {
        if (!this._footer) {
            this._footer = this.root.querySelector(".godjango-modal-footer");
        }
        return this._footer;
    }
}

function newModalEvent(action: "modal:open" | "modal:close", modal: Modal, event?: Event): ModalEvent {
    return new CustomEvent("modal:" + action, {
        detail: {
            action: action,
            modal: modal,
            originalEvent: event,
        }
    }) as ModalEvent;
}

type ModalConstructorOptions = {
    executeScriptsOnSet?: boolean;
    opened?: boolean;
    onOpen?: (event: Event) => void;
    onClose?: (event: Event) => void;
};

class Modal {
    private options: ModalConstructorOptions;
    private elements: Elements;
    private documentClickListener: (event: MouseEvent) => void;

    constructor(element: HTMLElement, opts: ModalConstructorOptions = {}) {
        let root = element as ModalElement;
        root.modal = this;
        root.dataset.modal = "true";
        this.options = opts;
        this.connect(root);

        if (this.options.opened) {
            this.open();
        }
    }
    
    actions(): ActionMap {
        return {
            close: this.close.bind(this),
            resize: (event: MouseEvent) => {
                this.elements.dialog.toggleAttribute("fullscreen");
            }
        }
    }

    executeAction(action: string) {
        const actions = this.actions();
        if (actions[action]) {
            actions[action]();
        } else {
            console.warn(`No action defined for ${action} in modal controls.`);
        }
    }

    connect(root: ModalElement) {
        this.elements = new Elements(root);
        this.elements.modal = (
            <div class="godjango-modal-wrapper">
                <dialog class="godjango-modal">
                    <div class="godjango-modal-controls" id="modal-controls">
                        <button class="godjango-modal-control godjango-modal-controls-modal-size" id="modal-controls-size" data-action="resize" type="button">
                            <svg xmlns="http://www.w3.org/2000/svg" fill="currentColor" class="godjango-modal-controls-icon" viewBox="0 0 16 16">
	                            {/* The MIT License (MIT) --> */}
	                            {/* Copyright (c) 2011-2024 The Bootstrap Authors --> */}
                                <path d="M0 3.5A1.5 1.5 0 0 1 1.5 2h13A1.5 1.5 0 0 1 16 3.5v9a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 0 12.5zM1.5 3a.5.5 0 0 0-.5.5v9a.5.5 0 0 0 .5.5h13a.5.5 0 0 0 .5-.5v-9a.5.5 0 0 0-.5-.5z"/>
                                <path d="M2 4.5a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1H3v2.5a.5.5 0 0 1-1 0zm12 7a.5.5 0 0 1-.5.5h-3a.5.5 0 0 1 0-1H13V8.5a.5.5 0 0 1 1 0z"/>
                            </svg>
                        </button>
                        <button class="godjango-modal-control godjango-modal-controls-modal-close" id="modal-controls-close" data-action="close" type="button">
                            <svg xmlns="http://www.w3.org/2000/svg" fill="currentColor" class="godjango-modal-controls-icon" viewBox="0 0 16 16">
	                            {/* The MIT License (MIT) --> */}
	                            {/* Copyright (c) 2011-2024 The Bootstrap Authors --> */}
                                <path d="M4.646 4.646a.5.5 0 0 1 .708 0L8 7.293l2.646-2.647a.5.5 0 0 1 .708.708L8.707 8l2.647 2.646a.5.5 0 0 1-.708.708L8 8.707l-2.646 2.647a.5.5 0 0 1-.708-.708L7.293 8 4.646 5.354a.5.5 0 0 1 0-.708"/>
                            </svg>
                        </button>
                    </div>
                    <div class="godjango-modal-header" id="modal-title">
                    </div>
                    <div class="godjango-modal-content" id="modal-content">
                    </div>
                    <div class="godjango-modal-footer" id="modal-footer">
                    </div>
                </dialog>
            </div>
        )

        this.elements.root.appendChild(
            this.elements.modal,
        );

        this.documentClickListener = (event: MouseEvent) => {
            console.debug("Modal document click listener", event.target);
            if (event.target instanceof HTMLElement && !this.elements.dialog.contains(event.target)) {
                this.close(event);
            }
        };

        const actors = this.elements.controls.querySelectorAll("[data-action]");
        const actions = this.actions();
        actors.forEach((actor: HTMLElement) => {
            const action = actor.dataset.action;
            if (action && actions[action]) {
                actor.addEventListener("click", (event: MouseEvent) => {
                    event.preventDefault();
                    actions[action](event);
                });
            } else {
                console.warn(`No action defined for ${action} in modal controls.`);
            }
        })
    }

    disconnect() {
        this.elements.root.modal = null;

        if (this.elements.modal) {
            this.elements.root.removeChild(this.elements.modal);
        }
    }

    open(event?: Event) {
        if (!this.elements.dialog) {
            console.warn("Modal dialog element not found.");
            return;
        }
        if (this.elements.dialog.open) {
            console.warn("Modal is already open.");
            return;
        }

        this.elements.dialog.showModal();
        this.elements.root.dispatchEvent(newModalEvent("modal:open", this, event));
        setTimeout(() => {
            this.elements.modal.addEventListener("click", this.documentClickListener);
        });
        if (this.options.onOpen) {
            this.options.onOpen(event);
        }
    }

    close(event?: Event) {
        if (!this.elements.dialog) {
            console.warn("Modal dialog element not found.");
            return;
        }
        if (!this.elements.dialog.open) {
            console.warn("Modal is not open.");
            return;
        }

        this.elements.dialog.close();
        this.elements.root.dispatchEvent(newModalEvent("modal:close", this, event));
        this.elements.root.removeEventListener("click", this.documentClickListener);
        if (this.options.onClose) {
            this.options.onClose(event);
        }
    }

    set controls(value: HTMLElement | string) {
        if (typeof value === "string") {
            this.executeAction(value);
        } else if (value instanceof HTMLElement) {
            this.elements.controls.innerHTML = "";
            this.elements.controls.appendChild(value);
        } else {
            console.warn("Invalid controls value:", value);
        }
    }

    set title(value: HTMLElement | string) {
        if (typeof value === "string") {
            this.elements.title.innerHTML = value;
        } else if (value instanceof HTMLElement) {
            this.elements.title.innerHTML = "";
            this.elements.title.appendChild(value);
        } else {
            console.warn("Invalid title value:", value);
        }
    }

    set content(value: HTMLElement | string) {
        if (typeof value === "string") {
            this.elements.content.innerHTML = value;
        } else if (value instanceof HTMLElement) {
            this.elements.content.innerHTML = "";
            this.elements.content.appendChild(value);
        } else {
            console.warn("Invalid content value:", value);
        }

        if (this.options.executeScriptsOnSet) {
            const scripts = this.elements.content.querySelectorAll("script");
            scripts.forEach(oldScript => {
                const newScript = document.createElement("script");
                newScript.dataset.initialized = "true";

                // Copy attributes
                [...oldScript.attributes].forEach(attr =>
                    newScript.setAttribute(attr.name, attr.value)
                );
            
                // Inline script content
                if (oldScript.textContent) {
                    newScript.textContent = oldScript.textContent;
                }
            
                // Replace the old script with the new one so the browser executes it
                oldScript.parentNode.replaceChild(newScript, oldScript);
            });
        }
    }

    get root(): ModalElement {
        return this.elements.root;
    }

    get dialog(): HTMLDialogElement {
        return this.elements.dialog;
    }

    get controls(): HTMLElement {
        return this.elements.controls;
    }

    get title(): HTMLElement {
        return this.elements.title;
    }

    get content(): HTMLElement {
        return this.elements.content;
    }
    
    get footer(): HTMLElement {
        return this.elements.footer;
    }
}

export {
    ModalEvent,
    ModalElement,
    Modal,
};
