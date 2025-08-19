const GoDjangoImagesIcon = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" class="bi bi-image" viewBox="0 0 16 16">
  <path stroke="none" stroke-width="0" d="M6.002 5.5a1.5 1.5 0 1 1-3 0 1.5 1.5 0 0 1 3 0"/>
  <path stroke="none" stroke-width="0" d="M2.002 1a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V3a2 2 0 0 0-2-2zm12 1a1 1 0 0 1 1 1v6.5l-3.777-1.947a.5.5 0 0 0-.577.093l-3.71 3.71-2.66-1.772a.5.5 0 0 0-.63.062L1.002 12V3a1 1 0 0 1 1-1z"/>
</svg>`;

class GoDjangoImagesTool {
    constructor({ data, api, config, block }) {
        if (!config) {
            config = {};
        }

        this.data = data || { ids: [] };
        this.data.ids = this.data.ids || [];

        this.api = api;
        this.block = block;
        this.config = config;
        this.imageWrapper = null;
    }

    get settings() {
        return [
            {
                name: window.i18n.gettext('Add Image'),
                icon: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" fill="currentColor" viewBox="0 0 16 16"><path fill-rule="evenodd" d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2"/></svg>`,
                onClick: function(api, block, data) {
                    block.chooser.open();
                },
            },
        ];
    }

    static get toolbox() {
      return {
            title: window.i18n.gettext('Images'),
            icon: GoDjangoImagesIcon,
        };
    }

    static newImage() {
        var imgWrapper = document.createElement('div');
        imgWrapper.classList.add("GoDjango-images-tool__image-wrapper");

        var img = document.createElement('img');
        img.classList.add("GoDjango-images-tool__image");
        imgWrapper.appendChild(img);
        imgWrapper.imageElement = img;
        return imgWrapper;
    }

    renderSettings() {
        const wrapper = document.createElement('div');
        wrapper.classList.add('GoDjango-images-tool__settings');

        for (const setting of this.settings) {
            let button = document.createElement('button');
            button.type = 'button';
            button.innerHTML = setting.icon;
            button.classList.add("cdx-settings-button");
            button.addEventListener(
                'click', () => setting.onClick(this.api, this, this.data),
            );
            wrapper.appendChild(button);
        }

        return wrapper;
    }

    validate(savedData) {
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
                this.imageWrapper.appendChild(image);
            },
            onClosed: () => {
                if (!this.data.ids || !this.data.ids.length) {
                    this.imageWrapper.remove();
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

        return this.imageWrapper
    }
  
    save(blockContent) {
        let idx = this.api.blocks.getCurrentBlockIndex();
        if (!blockContent && idx > -1) {
            this.api.blocks.delete(idx);
            return {
                ids: []
            };
        }

        const image = blockContent.querySelectorAll('img');
        var idList = [];
        image.forEach(img => {
            idList.push(img.dataset.id);
        });
        
        return {
            ids: idList
        };
    }
}

window.GoDjangoImagesTool = GoDjangoImagesTool;