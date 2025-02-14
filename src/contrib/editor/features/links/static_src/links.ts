import { API } from "@editorjs/editorjs";
import { PageChooserModal, PageMenuResponsePage } from "./modal";
import "./css/index.css";

const PageLinkIcon = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-link-45deg" viewBox="0 0 16 16">
  <path d="M4.715 6.542 3.343 7.914a3 3 0 1 0 4.243 4.243l1.828-1.829A3 3 0 0 0 8.586 5.5L8 6.086a1 1 0 0 0-.154.199 2 2 0 0 1 .861 3.337L6.88 11.45a2 2 0 1 1-2.83-2.83l.793-.792a4 4 0 0 1-.128-1.287z"/>
  <path d="M6.586 4.672A3 3 0 0 0 7.414 9.5l.775-.776a2 2 0 0 1-.896-3.346L9.12 3.55a2 2 0 1 1 2.83 2.83l-.793.792c.112.42.155.855.128 1.287l1.372-1.372a3 3 0 1 0-4.243-4.243z"/>
</svg>`;


type SavedData = {
    id: string,
    text: string,
}

type ConstructorObject = {
    data: SavedData | null,
    api: API,
    config: ToolConfig,
}

type ToolConfig = {
    pageListURL: string,
    pageListQueryVar: string,
}

class PageLinkTool {
    tag: string;
    tagClass: string;
    data: SavedData;
    api: API;
    config: ToolConfig;
    modal: PageChooserModal;
    private _state: boolean;

    wrapperTag: HTMLAnchorElement;
    button: HTMLButtonElement;
    actionsContainer: HTMLElement;
    chooseNewPageButton: HTMLButtonElement;

    constructor({ data, api, config }: ConstructorObject) {
        if (!config) {
            config = {} as ToolConfig;
        }

        this.tag = 'A';
        this.tagClass = 'page-link';

        this.data = data;
        this.api = api;
        this.config = config;

        if (
            !this.config.pageListURL ||
            !this.config.pageListQueryVar
        ) {
            throw new Error("PageLinkTool requires pageMenuURL, retrievePageURL, and pageIDVar in config");
        }
    }

    static get isInline() {
        return true;
    }

    static get sanitize() {
        return {
            a: true,
        };
    }

    set state(newState: boolean) {
        this._state = newState;
        this.button.classList.toggle(
            this.api.styles.inlineToolButtonActive, newState,
        );
    }

    get state() {
        return this._state;
    }

    validate(savedData: SavedData) {
        if (!savedData || !savedData.id || !savedData.text) {
            return false;
        }
        return savedData.id.trim() !== '' && savedData.text.trim() !== '';
    }

    surround(range: Range) {
        if (this.state) {
            this.unwrap(range);
            return;
        }

        this.modal.open();
        this.modal.onChosen = (page: PageMenuResponsePage) => {
            this.wrap(range, page);
        };
    }
    
    wrap(range: Range, page: PageMenuResponsePage) {
        let selectedText = range.extractContents();

        const previousWrapperTag = this.api.selection.findParentTag(this.tag);
        if (previousWrapperTag || previousWrapperTag && previousWrapperTag.querySelector(this.tag.toLowerCase())) {
            previousWrapperTag.remove();
        }

        this.wrapperTag = document.createElement(this.tag) as HTMLAnchorElement;
        this.wrapperTag.dataset.pageId = page.id;
        this.wrapperTag.href = page.url_path;

        this.wrapperTag.classList.add(this.tagClass);
        this.wrapperTag.appendChild(selectedText);
        range.insertNode(this.wrapperTag);

        this.api.selection.expandToTag(this.wrapperTag);
    }

    unwrap(range: Range) {
        const wrapperTag = this.api.selection.findParentTag(this.tag);
        const text = range.extractContents();
        wrapperTag.remove();
        range.insertNode(text);
    }
  
    render(){
        this.button = document.createElement('button');
        this.button.type = 'button';
        this.button.classList.add(this.api.styles.inlineToolButton);
        this.button.innerHTML = PageLinkIcon;
        this.modal = new PageChooserModal({
            pageListURL: this.config.pageListURL,
            pageListQueryVar: this.config.pageListQueryVar,
            translate: this.api.i18n.t.bind(this.api.i18n),
        })
        return this.button
    }
    
    checkState() {
        const wrapperTag = this.api.selection.findParentTag(this.tag, this.tagClass);

        this.state = !!wrapperTag;
    }

    save(blockContent: any) {
        const link = blockContent.querySelector('a');
        return {
            id: link.dataset.id,
            text: link.innerText,
        };
    }    
}

export {
    PageLinkTool,
};
