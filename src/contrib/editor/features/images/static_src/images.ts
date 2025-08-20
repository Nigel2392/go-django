import { API, BlockAPI, BlockToolConstructorOptions, ToolConfig } from "@editorjs/editorjs";
import { Chooser } from "../../../../admin/chooser/static_src/chooser/chooser";


const GoDjangoImagesIcon = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" class="bi bi-image" viewBox="0 0 16 16">
	<!-- The MIT License (MIT) -->
	<!-- Copyright (c) 2011-2024 The Bootstrap Authors -->
    <path stroke="none" stroke-width="0" d="M6.002 5.5a1.5 1.5 0 1 1-3 0 1.5 1.5 0 0 1 3 0"/>
    <path stroke="none" stroke-width="0" d="M2.002 1a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V3a2 2 0 0 0-2-2zm12 1a1 1 0 0 1 1 1v6.5l-3.777-1.947a.5.5 0 0 0-.577.093l-3.71 3.71-2.66-1.772a.5.5 0 0 0-.63.062L1.002 12V3a1 1 0 0 1 1-1z"/>
</svg>`;

const iconStretch = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-arrows-fullscreen" viewBox="0 0 16 16">
	<!-- The MIT License (MIT) -->
	<!-- Copyright (c) 2011-2024 The Bootstrap Authors -->
    <path fill-rule="evenodd" d="M5.828 10.172a.5.5 0 0 0-.707 0l-4.096 4.096V11.5a.5.5 0 0 0-1 0v3.975a.5.5 0 0 0 .5.5H4.5a.5.5 0 0 0 0-1H1.732l4.096-4.096a.5.5 0 0 0 0-.707m4.344 0a.5.5 0 0 1 .707 0l4.096 4.096V11.5a.5.5 0 1 1 1 0v3.975a.5.5 0 0 1-.5.5H11.5a.5.5 0 0 1 0-1h2.768l-4.096-4.096a.5.5 0 0 1 0-.707m0-4.344a.5.5 0 0 0 .707 0l4.096-4.096V4.5a.5.5 0 1 0 1 0V.525a.5.5 0 0 0-.5-.5H11.5a.5.5 0 0 0 0 1h2.768l-4.096 4.096a.5.5 0 0 0 0 .707m-4.344 0a.5.5 0 0 1-.707 0L1.025 1.732V4.5a.5.5 0 0 1-1 0V.525a.5.5 0 0 1 .5-.5H4.5a.5.5 0 0 1 0 1H1.732l4.096 4.096a.5.5 0 0 1 0 .707"/>
</svg>`;

const iconUnstretch = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-fullscreen-exit" viewBox="0 0 16 16">
	<!-- The MIT License (MIT) -->
	<!-- Copyright (c) 2011-2024 The Bootstrap Authors -->
    <path d="M5.5 0a.5.5 0 0 1 .5.5v4A1.5 1.5 0 0 1 4.5 6h-4a.5.5 0 0 1 0-1h4a.5.5 0 0 0 .5-.5v-4a.5.5 0 0 1 .5-.5m5 0a.5.5 0 0 1 .5.5v4a.5.5 0 0 0 .5.5h4a.5.5 0 0 1 0 1h-4A1.5 1.5 0 0 1 10 4.5v-4a.5.5 0 0 1 .5-.5M0 10.5a.5.5 0 0 1 .5-.5h4A1.5 1.5 0 0 1 6 11.5v4a.5.5 0 0 1-1 0v-4a.5.5 0 0 0-.5-.5h-4a.5.5 0 0 1-.5-.5m10 1a1.5 1.5 0 0 1 1.5-1.5h4a.5.5 0 0 1 0 1h-4a.5.5 0 0 0-.5.5v4a.5.5 0 0 1-1 0z"/>
</svg>`;

const textAddImage = window.i18n.gettext('Add %s', window.i18n.gettext('Image'));
const textStretchImage = window.i18n.gettext('Stretch %s', window.i18n.gettext('Images'));
const textUnstretchImage = window.i18n.gettext('Unstretch %s', window.i18n.gettext('Images'));

type GoDjangoImagesToolData = {
    ids: string[];
    stretched: boolean;
};

class ImageWrapperElement extends HTMLDivElement {
    imageElement: HTMLImageElement;
};

type Setting = {
    name: string;
    icon: string;
    onClick: (api: API, block: GoDjangoImagesTool, data: GoDjangoImagesToolData, element: SettingsButton) => void;
    onInit?: (api: API, block: GoDjangoImagesTool, data: GoDjangoImagesToolData, element: SettingsButton) => void;
};

class SettingsButton {
    button: HTMLButtonElement;

    constructor(button: HTMLButtonElement) {
        this.button = button;
    }
    
    get classList() {
        return this.button.classList;
    }

    set icon(icon: string) {
        let i = this.button.querySelector("svg");
        if (i) {
            i.outerHTML = icon;
        } else {
            this.button.innerHTML = icon;
        }
    }
}

class GoDjangoImagesTool {

    data: GoDjangoImagesToolData;
    api: API;
    block: BlockAPI;
    config: ToolConfig;
    imageWrapper: HTMLDivElement | null;
    chooser: Chooser;

    constructor({ data, api, config, block }: BlockToolConstructorOptions<GoDjangoImagesToolData, ToolConfig>) {
        if (!config) {
            config = {};
        }
        
        let stretched = data.stretched;
        if ("defaultStretched" in config && !("stretched" in data)) {
            stretched = config.defaultStretched;
        }

        this.data = data || { ids: [], stretched: stretched };
        this.data.ids = this.data.ids || [];

        this.api = api;
        this.block = block;
        this.config = config;
        this.imageWrapper = null;
    }

    get settings(): Setting[] {
        return [
            {
                name: textAddImage,
                icon: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" fill="currentColor" viewBox="0 0 16 16">
                    <path fill-rule="evenodd" d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2"/>
                </svg>`,
                onClick: function(api: API, block: GoDjangoImagesTool, data: GoDjangoImagesToolData, element: SettingsButton) {
                    block.chooser.open();
                },
            },
            {
                name: this.data.stretched ? textUnstretchImage : textStretchImage,
                icon: this.data.stretched ? iconStretch: iconUnstretch,
                onClick: function(api: API, block: GoDjangoImagesTool, data: GoDjangoImagesToolData, element: SettingsButton) {
                    block.data.stretched = !data.stretched;
                    block.block.stretched = block.data.stretched;

                    this.onInit(api, block, data, element);
                },
                onInit: function(api: API, block: GoDjangoImagesTool, data: GoDjangoImagesToolData, element: SettingsButton) {
                    if (block.data.stretched) {
                        element.classList.add(api.styles.settingsButtonActive);
                        element.icon = iconUnstretch;
                        api.tooltip.onHover(element.button, textUnstretchImage);
                    } else {
                        element.classList.remove(api.styles.settingsButtonActive);
                        element.icon = iconStretch;
                        api.tooltip.onHover(element.button, textStretchImage);
                    }
                }
            },
        ];
    }

    static get toolbox() {
      return {
            title: window.i18n.gettext('Images'),
            icon: GoDjangoImagesIcon,
        };
    }

    static newImage(): ImageWrapperElement {
       let imgWrapper = document.createElement('div') as ImageWrapperElement;
       imgWrapper.classList.add('GoDjango-images-tool__image-wrapper');

       var img = document.createElement('img');
       img.classList.add('GoDjango-images-tool__image');
       imgWrapper.appendChild(img);
       imgWrapper.imageElement = img;
       return imgWrapper;
    }

    renderSettings() {
        const wrapper = document.createElement('div');
        wrapper.classList.add('GoDjango-images-tool__settings');

        for (const setting of this.settings) {
            let elem = new SettingsButton(document.createElement('button'));
            let button = elem.button;
            button.type = 'button';
            button.classList.add(this.api.styles.settingsButton);

            elem.icon = setting.icon;

            this.api.tooltip.onHover(elem.button, setting.name);

            if (setting.onInit) {
                setting.onInit(this.api, this, this.data, elem);
            }

            button.addEventListener(
                'click', () => setting.onClick.bind(setting)(this.api, this, this.data, elem)
            );
            wrapper.appendChild(button);
        }

        return wrapper;
    }

    validate(savedData: GoDjangoImagesToolData) {
        if (!savedData.ids) {
            return false;
        }
        return savedData.ids.length > 0;
    }
  
    render(){
        this.imageWrapper = document.createElement('div');
        this.imageWrapper.classList.add('GoDjango-images-tool');

        this.chooser = new window.Chooser({
            title:   window.i18n.gettext('Select an image'),
            listurl: this.config.chooserURL,
            onChosen: (value, previewText, data) => {
                this.data.ids.push(value);
                let image = GoDjangoImagesTool.newImage();
                image.imageElement.src = this.config.serveUrl.replace("<<id>>", value);
                image.imageElement.dataset.id = value;
                this.imageWrapper?.appendChild(image);
            },
            onClosed: () => {
                if (!this.data.ids || !this.data.ids.length) {
                    this.imageWrapper?.remove();
                    this.api.blocks.delete(this.api.blocks.getCurrentBlockIndex());
                }
            }
        })

        if (this.data.ids && this.data.ids.length > 0) {
            for (const id of this.data.ids) {
                let image = GoDjangoImagesTool.newImage();
                image.imageElement.src = this.config.serveUrl.replace("<<id>>", id);
                image.imageElement.dataset.id = id;
                this.imageWrapper.appendChild(image);
            }
        } else {
            this.chooser.open()
        }

        setTimeout(() => {
            if (this.data.stretched) {
                this.block.stretched = true;
            } 
        });

        return this.imageWrapper
    }

    save(blockContent: HTMLElement) {
        let idx = this.api.blocks.getCurrentBlockIndex();
        if (!blockContent && idx > -1) {
            this.api.blocks.delete(idx);
            return {
                ids: [],
                stretched: this.block.stretched
            };
        }

        const image = blockContent.querySelectorAll('img');
        var idList: string[] = [];
        image.forEach(img => {
            idList.push(img.dataset.id as string);
        });
        
        return {
            ids: idList,
            stretched: this.block.stretched || false
        };
    }
}

declare global {
    interface Window {
        GoDjangoImagesTool: typeof GoDjangoImagesTool;
        i18n: {
            gettext: (key: string, ...args: any) => string;
        };
    }
}

window.GoDjangoImagesTool = GoDjangoImagesTool;
