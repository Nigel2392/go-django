import { jsx } from "./jsx";


type PageMenuResponse = {
    header: {
        text: string,
        url: string,
    },
    parent_item?: PageMenuResponsePage,
    items: PageMenuResponsePage[],
}

type PageMenuResponsePage = {
    id: string,
    title: string,
    numchild: number,
    depth: number,
}

type ModalOptions = {
    menuURL: string,
    menuQueryVar: string,
    openedByDefault?: boolean,
    translate?: (key: string) => string,
    pageChosen?: (page: PageMenuResponsePage) => void,
    modalOpen?: (modal: PageChooserModal) => void,
    modalClose?: (modal: PageChooserModal) => void,
    modalDelete?: (modal: PageChooserModal) => void,
}

class PageChooserModal {
    private elements: {
        overlay: HTMLElement;
        modal: HTMLElement;
        loader: HTMLElement;
        close: HTMLElement;
        content: HTMLElement;
        error: HTMLElement;
    }
    private opts: ModalOptions;
    private _state: PageMenuResponsePage | null;

    constructor(opts: ModalOptions) {
        this.elements = {
            overlay: null,
            modal: null,
            loader: null,
            close: null,
            content: null,
            error: null,
        };
        this.opts = opts;
    }

    private initChooser() {
        this.elements.overlay = <div class="page-link-modal-overlay">
            <div class="page-link-modal">
                <div class="page-link-modal-controls">
                    <button class="page-link-modal-control page-link-modal-close" type="button">
                        <span className="page-link-modal-close-x">×</span>
                    </button>
                </div>

                <div class="page-link-modal-heading">
                    <h2>{this.opts.translate("Choose a Page")}</h2>
                </div>
                
                <div class="page-link-modal-loader" role="status" style="margin-bottom:2%;"></div>

                <div class="page-link-modal-error"></div>

                <div class="page-link-modal-content">
                    
                </div>
            </div>
        </div>;

        this.elements.modal = this.elements.overlay.querySelector('.page-link-modal');
        this.elements.loader = this.elements.modal.querySelector('.page-link-modal-loader');
        this.elements.close = this.elements.modal.querySelector('.page-link-modal-close');
        this.elements.content = this.elements.modal.querySelector('.page-link-modal-content');
        this.elements.error = this.elements.modal.querySelector('.page-link-modal-error');

        this.elements.close.addEventListener('click', () => {
            this.elements.overlay.remove();
        });

        if (this.opts.openedByDefault) {
            this.elements.overlay.hidden = false;
            if (this.opts.modalOpen) {
                this.opts.modalOpen(this);
            }
        }

        document.body.appendChild(this.elements.overlay);

        this.modalError(null);
        this.loadPageList();        
    }

        
    private async loadPageList() {
        this.modalError(null);
        this.elements.loader.hidden = false;
        fetch(this.opts.menuURL)
        .then(response => response.json())
        .then((data: PageMenuResponse) => {
            this.elements.loader.hidden = true;

            if (data.items.length == 0) {
                this.modalError(this.opts.translate("No pages found"));
                return;
            }

            for (let page of data.items) {

                const pageListItem = <div class="page-link-modal-page" data-page-id={page.id}>
                    <div className="page-link-modal-page-heading">
                        {page.title}
                    </div>
                </div>;

                if (page.numchild > 0) {
                    pageListItem.appendChild(<div class="page-link-modal-page-children">
                        {page.numchild.toString()} {this.opts.translate("children")}
                    </div>)
                }

                var heading = pageListItem.querySelector('.page-link-modal-page-heading');
                heading.addEventListener('click', () => {
                    this._state = page;
                    if (this.opts.pageChosen) {
                        this.opts.pageChosen(page);
                    }
                    let anim = this.elements.overlay.animate([
                        {opacity: 1},
                        {opacity: 0},
                    ], {
                        duration: 200,
                        fill: "forwards",
                    });
                    anim.onfinish = () => {
                        this.elements.overlay.hidden = true;
                    }
                });

                this.elements.content.appendChild(pageListItem);
            }
        })
        .catch(error => {
            console.error("Error fetching menu items", error);
            this.elements.loader.hidden = true;
            this.modalError(this.opts.translate(
                "Error fetching menu items, please try again later or contact an administrator"
            ));
        });
    }

    get state() {
        return this._state;
    }

    set onOpen(callback: (modal: PageChooserModal) => void) {
        this.opts.modalOpen = callback;
    }

    set onClose(callback: (modal: PageChooserModal) => void) {
        this.opts.modalClose = callback;
    }

    set onDelete(callback: (modal: PageChooserModal) => void) {
        this.opts.modalDelete = callback;
    }

    set onChosen(callback: (page: PageMenuResponsePage) => void) {
        this.opts.pageChosen = callback
    }

    open() {
        if (!this.elements.overlay) {
            this.initChooser();
        } else {
            this.elements.overlay.hidden = false;
        }

        if (this.opts.modalOpen) {
            this.opts.modalOpen(this);
        }
    }

    close() {
        this.elements.overlay.hidden = true;

        if (this.opts.modalClose) {
            this.opts.modalClose(this);
        }
    }

    delete() {
        if (this.elements.overlay) {
            this.elements.overlay.remove();
        }

        if (this.opts.modalDelete) {
            this.opts.modalDelete(this);
        }
    }

    modalError(message: string | null) {
        if (message == "" || message == null) {
            this.elements.error.innerHTML = "";
        }

        this.elements.error.innerText = message;
    }
}

export {
    PageChooserModal,
    ModalOptions,
    PageMenuResponse,
    PageMenuResponsePage,
}