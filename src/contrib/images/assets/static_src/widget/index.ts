class ImageChooserWidgetConfig {
    selector: string;
    serveURL: string;
    uploadURL: string;
    listURL: string;
    deleteURL: string;

    get canUpload(): boolean {
        return this.uploadURL !== '';
    }
    get canList(): boolean {
        return this.listURL !== '';
    }
    get canDelete(): boolean {
        return this.deleteURL !== '';
    }

    constructor(selector: string, serveURL: string, uploadURL: string, listURL: string, deleteURL: string) {
        this.selector = selector;
        this.serveURL = serveURL;
        this.uploadURL = uploadURL;
        this.listURL = listURL;
        this.deleteURL = deleteURL;
    }
}

class ImageChooserWidgetModal extends EventTarget {
    image: HTMLImageElement;
    container: HTMLElement;
    fileInput: HTMLInputElement;
    titleInput: HTMLInputElement;

    constructor(container: HTMLElement) {
        if (!container) {
            throw new Error("Container element is required for ImageChooserWidgetModal.");
        }

        super();

        this.container = container;
        this.image = this.container.querySelector('img') as HTMLImageElement;
        this.fileInput = this.container.querySelector('input[type="file"]') as HTMLInputElement;
        this.titleInput = this.container.querySelector('input[type="text"]') as HTMLInputElement;
        this.init();
    }

    init() {
        // Initialize the modal
    }

    onUpdate(callback: EventListenerOrEventListenerObject ) {
        this.addEventListener('update', callback);
    }

    onClear(callback: EventListenerOrEventListenerObject ) {
        this.addEventListener('clear', callback);
    }
}

class ImageChooserWidget {
    container: HTMLElement;
    input: HTMLInputElement;
    modal: ImageChooserWidgetModal;
    config: ImageChooserWidgetConfig;

    constructor(config: ImageChooserWidgetConfig) {
        this.container = document.querySelector(config.selector) as HTMLElement;
        this.modal = new ImageChooserWidgetModal(this.container);
        this.config = config;
    }
    
}

export { ImageChooserWidget, ImageChooserWidgetConfig };