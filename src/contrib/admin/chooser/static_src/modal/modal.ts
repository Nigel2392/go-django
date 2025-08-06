type ModalEvent = Event & { detail: { action: "modal:open" | "modal:close", modal: Modal }};

type ModalElement = HTMLElement & { Modal?: Modal, dataset: { Modal: string } };

type Elements = {
    root: ModalElement;
    dialog: HTMLDialogElement;
    title: HTMLElement;
    errors: HTMLElement;
    content: HTMLElement;
    footer: HTMLElement;
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

class Modal {
    elements: Elements;
    
    constructor(element: HTMLElement) {
        let root = element as ModalElement;

        root.Modal = this;
        root.dataset.Modal = "true";

        this.elements = Modal.newElements(root);
        this.tryClose();
    }

    static newElements(root: ModalElement): Elements {
        return {
            root: root,
            dialog: root.querySelector("dialog") as HTMLDialogElement,
            title: root.querySelector("#modal-title") as HTMLElement,
            errors: root.querySelector("#modal-errors") as HTMLElement,
            content: root.querySelector("#modal-content") as HTMLElement,
            footer: root.querySelector("#modal-footer") as HTMLElement,
        };
    }

    disconnect() {
        this.elements.root.Modal = null;
        this.elements.dialog = null;
        this.elements.root.remove();
    }

    tryClose() {
        var modals = document.querySelectorAll("[data-controller='modal']");
        if (modals.length == 1) {
            return;
        }

        console.warn(
            "Multiple modal controllers found on the page. This may cause unexpected behavior.",
        );

        modals.forEach((modal: ModalElement) => {
            if (modal !== this.elements.root) {
                modal.Modal?.disconnect();
            }
        });
    }

    open(event?: Event) {
        this.elements.dialog.showModal();
        this.elements.root.dispatchEvent(newModalEvent("modal:open", this, event));
    }

    close(event?: Event) {
        this.elements.dialog.close();
        this.elements.root.dispatchEvent(newModalEvent("modal:close", this, event));
    }


    set root(value: ModalElement | HTMLElement | null) {
        if (value === null) {
            this.disconnect();
            return;
        }

        var rootParent = this.elements.root.parentNode;
        if (rootParent) {
            rootParent.replaceChild(value, this.elements.root);
        }
        this.elements.root = value as ModalElement;
        this.elements.root.Modal = this;
        this.elements.root.dataset.Modal = "true";
        this.elements = Modal.newElements(this.elements.root);
    }

    set dialog(value: HTMLDialogElement | HTMLElement | null) {
        if (value === null) {
            this.disconnect();
            return;
        }
        
        var parent = this.elements.dialog.parentNode;
        if (parent) {
            parent.replaceChild(value, this.elements.dialog);
        }
        this.elements.dialog = value as HTMLDialogElement;
    }

    set title(value: HTMLElement | string | null) {
        if (value === null) {
            this.elements.title.textContent = "";
            return;
        }

        if (typeof value === "string") {
            this.elements.title.textContent = value;
        } else {
            var parent = this.elements.title.parentNode;
            if (parent) {
                parent.replaceChild(value, this.elements.title);
            }
            this.elements.title = value as HTMLElement;
        }
    }

    set errors(value: HTMLElement | string[] | null) {
        if (value === null) {
            this.elements.errors.textContent = "";
            return;
        }

        if (Array.isArray(value)) {
            this.elements.errors.innerHTML = "";
            value.forEach((error) => {
                var errorElement = document.createElement("div");
                errorElement.textContent = error;
                this.elements.errors.appendChild(errorElement);
            });
            return;
        }

        var parent = this.elements.errors.parentNode;
        if (parent) {
            parent.replaceChild(value, this.elements.errors);
        }
        this.elements.errors = value as HTMLElement;
    }

    set content(value: HTMLElement | string | null) {
        if (value === null) {
            this.elements.content.textContent = "";
            return;
        }

        if (typeof value === "string") {
            this.elements.content.innerHTML = value;
        } else {
            var parent = this.elements.content.parentNode;
            if (parent) {
                parent.replaceChild(value, this.elements.content);
            }
            this.elements.content = value as HTMLElement;
        }
    }

    set footer(value: HTMLElement | string | null) {
        if (value === null) {
            this.elements.footer.textContent = "";
            return;
        }

        if (typeof value === "string") {
            this.elements.footer.innerHTML = value;
        } else {
            var parent = this.elements.footer.parentNode;
            if (parent) {
                parent.replaceChild(value, this.elements.footer);
            }
            this.elements.footer = value as HTMLElement;
        }
    }

    get root(): ModalElement {
        return this.elements.root;
    }

    get dialog(): HTMLDialogElement {
        return this.elements.dialog;
    }

    get title(): HTMLElement {
        return this.elements.title;
    }

    get errors(): HTMLElement {
        return this.elements.errors;
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
