
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

class ImageChooserWidget {
    container: HTMLElement;
    image: HTMLImageElement;
    fileInput: HTMLInputElement;
    titleInput: HTMLInputElement;
    config: ImageChooserWidgetConfig;

    constructor(config: ImageChooserWidgetConfig) {
        this.container = document.querySelector(config.selector) as HTMLElement;
        this.image = this.container.querySelector('img') as HTMLImageElement;
        this.fileInput = this.container.querySelector('input[type="file"]') as HTMLInputElement;
        this.titleInput = this.container.querySelector('input[type="text"]') as HTMLInputElement;
    }
}

