import { jsx } from "./jsx";


type PageMenuResponse = {
    parent_item?: PageMenuResponsePage,
    items: PageMenuResponsePage[],
}

type PageMenuResponsePage = {
    id: string,
    title: string,
    numchild: number,
    url_path: string,
    depth: number,
}

type ModalOptions = {
    pageListURL: string,
    pageListQueryVar: string,
    openedByDefault?: boolean,
    translate?: (key: string) => string,
    pageChosen?: (page: PageMenuResponsePage) => void,
    modalOpen?: (modal: PageChooserModal) => void,
    modalClose?: (modal: PageChooserModal) => void,
    modalDelete?: (modal: PageChooserModal) => void,
}

const SVGGetParent = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-arrow-return-left" viewBox="0 0 16 16">
    <path fill-rule="evenodd" d="M14.5 1.5a.5.5 0 0 1 .5.5v4.8a2.5 2.5 0 0 1-2.5 2.5H2.707l3.347 3.346a.5.5 0 0 1-.708.708l-4.2-4.2a.5.5 0 0 1 0-.708l4-4a.5.5 0 1 1 .708.708L2.707 8.3H12.5A1.5 1.5 0 0 0 14 6.8V2a.5.5 0 0 1 .5-.5"/>
</svg>`

const SVGGetChildren = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-box-arrow-in-right" viewBox="0 0 16 16">
    <path fill-rule="evenodd" d="M6 3.5a.5.5 0 0 1 .5-.5h8a.5.5 0 0 1 .5.5v9a.5.5 0 0 1-.5.5h-8a.5.5 0 0 1-.5-.5v-2a.5.5 0 0 0-1 0v2A1.5 1.5 0 0 0 6.5 14h8a1.5 1.5 0 0 0 1.5-1.5v-9A1.5 1.5 0 0 0 14.5 2h-8A1.5 1.5 0 0 0 5 3.5v2a.5.5 0 0 0 1 0z"/>
    <path fill-rule="evenodd" d="M11.854 8.354a.5.5 0 0 0 0-.708l-3-3a.5.5 0 1 0-.708.708L10.293 7.5H1.5a.5.5 0 0 0 0 1h8.793l-2.147 2.146a.5.5 0 0 0 .708.708z"/>
</svg>`

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

    private _try(errorMessage: string, func: any, ...args: any[]) {
        try {
            return func(...args);
        } catch (e) {
            this.modalError(errorMessage);
        }
    }

    private async loadPages(id: number | string = null, get_parent: boolean = false) {
        // Get ID, default to current state's ID
        // if no ID is provided and no state is set
        // then we default to an empty string, this
        // will fetch all root pages
        if (id == null) {
            id = this.state ? this.state.id : "";
        }

        // Setup
        this.modalError(null);
        this.elements.loader.hidden = false;

        // Build URL
        const query = new URLSearchParams({
            [this.opts.pageListQueryVar]: id.toString(),
            get_parent: get_parent.toString(),
        });
        const url = this.opts.pageListURL + "?" + query.toString();
        
        // Make request
        const response = await this._try("Error fetching menu items", fetch, url);
        if (!response.ok) {
            this.modalError(this.opts.translate("Error fetching menu items"));
            return;
        }

        // Parse response
        const data = await this._try("Error parsing response", response.json.bind(response));
        this.elements.loader.hidden = true;
        return data;
    }
        
    private async loadPageList(pageId: number | string = null, get_parent: boolean = false) {
        this.modalError(null);
        this.elements.loader.hidden = false;
        let data = await this.loadPages(pageId, get_parent);
        if (!data) {
            return;
        }

        this.elements.content.innerHTML = "";
        if (data.items.length == 0 && !data.parent_item) {
            this.modalError(this.opts.translate("No pages found"));
            return;
        }


        if (data.parent_item) {
            const navUp = <div class="page-link-modal-parent-page-pageup"></div>
            navUp.innerHTML = SVGGetParent;

            const parentItem = <div class="page-link-modal-parent-page" data-page-id={data.parent_item.id} data-depth={data.parent_item.depth}>
                { navUp }
                <div class="page-link-modal-parent-page-heading">
                    {data.parent_item.title}
                </div>
            </div>;

            navUp.addEventListener('click', () => {
                this.loadPageList(data.parent_item.id, true);
                this._state = data.parent_item;
            });

            this.elements.content.appendChild(parentItem);

            if (data.items.length == 0) {

                this.elements.content.appendChild(<div class="page-link-modal-page">
                    <div class="page-link-modal-page-heading">
                        {this.opts.translate("No live child pages found")}
                    </div>
                </div>);
            }
        }


        for (let page of data.items) {
            
            const pageListItem = <div class="page-link-modal-page" data-page-id={page.id} data-depth={page.depth}>
                <div className="page-link-modal-page-heading">
                    {page.title}
                </div>
            </div>;

            if (page.numchild > 0) {
                let navDown = <div class="page-link-modal-page-down"></div>;
                navDown.innerHTML = SVGGetChildren;

                navDown.addEventListener('click', () => {
                    this.loadPageList(page.id);
                    this._state = page;
                });
                
                pageListItem.appendChild(navDown);
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
        this.elements.loader.hidden = true;
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