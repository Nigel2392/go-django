import { API, BlockAPI, BlockToolConstructorOptions, ToolConfig } from "@editorjs/editorjs";

const GoDjangoImageIcon = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" class="bi bi-image" viewBox="0 0 16 16">
	<!-- The MIT License (MIT) -->
	<!-- Copyright (c) 2011-2024 The Bootstrap Authors -->
    <path stroke="none" stroke-width="0" d="M6.002 5.5a1.5 1.5 0 1 1-3 0 1.5 1.5 0 0 1 3 0"/>
    <path stroke="none" stroke-width="0" d="M2.002 1a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V3a2 2 0 0 0-2-2zm12 1a1 1 0 0 1 1 1v6.5l-3.777-1.947a.5.5 0 0 0-.577.093l-3.71 3.71-2.66-1.772a.5.5 0 0 0-.63.062L1.002 12V3a1 1 0 0 1 1-1z"/>
</svg>`;

type GoDjangoImageToolData = {
    id: string;
    caption: string;
};

type ConstructorData = {
    data: GoDjangoImageToolData;
    api: API;
    config: ToolConfig;
    block: BlockAPI;
};

class GoDjangoImageTool {
    data: GoDjangoImageToolData;
    api: API;
    block: BlockAPI;
    config: ToolConfig;
    imageWrapper: HTMLDivElement | null;
    image: HTMLImageElement;
    chooser: any;

    constructor({ data, api, config, block }: BlockToolConstructorOptions<GoDjangoImageToolData, ToolConfig>) {
        if (!config) {
            config = {};
        }

        this.data = data;
        this.api = api;
        this.block = block;
        this.config = config;
        this.imageWrapper = null;
    }

    static get toolbox() {
      return {
            title: window.i18n.gettext('Image'),
            icon: GoDjangoImageIcon,
        };
    }

    validate(savedData: GoDjangoImageToolData) {
        return !!savedData.id;
    }
  
    render(){
        this.imageWrapper = document.createElement('div');
        this.imageWrapper.classList.add('GoDjango-image-tool');

        this.image = document.createElement('img');
        this.image.classList.add(
            this.api.styles.block,
            "GoDjango-image-tool__image"
        );

        this.chooser = new window.Chooser({
            title:   window.i18n.gettext('Select an image'),
            listurl: this.config.chooserURL,
            onChosen: (value, previewText, data) => {
                this.data.id = value;
                this.data.caption = data.caption;
                this.image.src = this.config.serveUrl.replace("<<id>>", value);
                this.image.alt = data.caption;
                this.image.dataset.id = value;
            },
            onClosed: () => {
                if (!this.data.id) {
                    this.imageWrapper?.remove();
                    this.api.blocks.delete(this.api.blocks.getCurrentBlockIndex());
                }
            }
        })

        if (this.data.id) {
            this.image.src = this.config.serveUrl.replace("<<id>>", this.data.id)
            this.image.alt = this.data.caption || '';
            this.image.dataset.id = this.data.id;
            this.image.dataset.caption = this.data.caption || '';
        } else {
            this.chooser.open()
        }

        this.image.style.cursor = 'pointer';
        this.api.tooltip.onHover(
            this.image,
            window.i18n.gettext('CTRL + Click to change image'),
            {
                placement: 'top',
            },
        );
        this.image.addEventListener('click', (e) => {
            if (e.ctrlKey) {
                this.chooser.open();
            }
        });

        this.imageWrapper.appendChild(this.image);

        return this.imageWrapper
    }

    save(blockContent: HTMLElement) {
        let idx = this.api.blocks.getCurrentBlockIndex();
        if (!blockContent && idx > -1) {
            this.api.blocks.delete(idx);
            return;
        }

        const image = blockContent.querySelector('img');
        return {
            id: image.dataset.id,
            caption: image.alt,
        };
    }
}

declare global {
    interface Window {
        GoDjangoImageTool: typeof GoDjangoImageTool;
    }
}

window.GoDjangoImageTool = GoDjangoImageTool;