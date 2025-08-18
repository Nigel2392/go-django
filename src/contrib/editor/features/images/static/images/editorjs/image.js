const GoDjangoImageIcon = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" class="bi bi-image" viewBox="0 0 16 16">
  <path stroke="none" stroke-width="0" d="M6.002 5.5a1.5 1.5 0 1 1-3 0 1.5 1.5 0 0 1 3 0"/>
  <path stroke="none" stroke-width="0" d="M2.002 1a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V3a2 2 0 0 0-2-2zm12 1a1 1 0 0 1 1 1v6.5l-3.777-1.947a.5.5 0 0 0-.577.093l-3.71 3.71-2.66-1.772a.5.5 0 0 0-.63.062L1.002 12V3a1 1 0 0 1 1-1z"/>
</svg>`;

const GoDjangoImagesIcon = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-images" viewBox="0 0 16 16">
  <path d="M4.502 9a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3"/>
  <path d="M14.002 13a2 2 0 0 1-2 2h-10a2 2 0 0 1-2-2V5A2 2 0 0 1 2 3a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2v8a2 2 0 0 1-1.998 2M14 2H4a1 1 0 0 0-1 1h9.002a2 2 0 0 1 2 2v7A1 1 0 0 0 15 11V3a1 1 0 0 0-1-1M2.002 4a1 1 0 0 0-1 1v8l2.646-2.354a.5.5 0 0 1 .63-.062l2.66 1.773 3.71-3.71a.5.5 0 0 1 .577-.094l1.777 1.947V5a1 1 0 0 0-1-1z"/>
</svg>`;

function getCsrfToken() {
    var formInput = document.querySelector('input[name="csrf_token"]');
    if (formInput) {
        return formInput.value;
    }
    return null;
}

const STATUS_SUCCESS = 'success';
const STATUS_ERROR = 'error';

class GoDjangoImageTool {
    constructor({ data, api, config, block }) {
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
            title: 'Image',
            icon: GoDjangoImageIcon,
        };
    }

    validate(savedData) {
        return savedData.filePath && savedData.filePath.length > 0 && savedData.id;
    }
  
    render(){
        this.imageWrapper = document.createElement('div');
        this.imageWrapper.classList.add('GoDjango-image-tool');

        this.image = document.createElement('img');
        this.image.classList.add(
            this.api.styles.block,
            "GoDjango-image-tool__image"
        );

        const throwError = (message) => {
            let element = document.createElement('p');
            element.textContent = message;
            element.style.color = 'red';
            this.imageWrapper.appendChild(element);
            return;
        }

        function createFileInput(wrapper) {
            var fileInput = document.createElement('input');
            fileInput.type = 'file';
            fileInput.accept = 'image/*';
            fileInput.style.display = 'none';
            wrapper.appendChild(fileInput);
            fileInput.addEventListener('change', async (e) => {

                let csrfTokenResponse = await fetch(`${this.config.uploadUrl}`, {
                    method: 'GET',
                })
                let csrfTokenData = await csrfTokenResponse.json();
                let csrftoken = csrfTokenData.csrf_token;

                var formData = new FormData();
                formData.append('file', fileInput.files[0]);
                formData.append('csrf_token', csrftoken);

                fetch(this.config.uploadUrl, {
                    method: 'POST',
                    body: formData,
                })
                .then(response => response.json())
                .then(data => {

                    if (data.status !== STATUS_SUCCESS) {
                        let message = this.api.i18n.t('Failed to upload image');
                        if (data.message) {
                            message = data.message;
                        }
                        throwError(message);
                        return;
                    }

                    this.data.id = data.id;
                    this.data.caption = data.caption;
                    this.data.filePath = data.filePath;
                    this.image.src = `${this.config.serveUrl}/${data.filePath}`
                    this.image.dataset.id = data.id;
                    this.image.dataset.filePath = data.filePath;
                    fileInput.remove();

                }).catch((error) => {
                    let message = this.api.i18n.t('Failed to upload image');
                    if (error.message) {
                        message = error.message;
                    }
                    throwError(message);
                })

            });
            return fileInput;
        }

        createFileInput = createFileInput.bind(this);
        if (this.data.filePath) {
            this.image.src = this.config.serveUrl.replace("<<id>>", this.data.id)
            this.image.dataset.id = this.data.id;
            this.image.dataset.caption = this.data.caption || '';
            this.image.dataset.filePath = this.data.filePath;
        } else {
            var fileInput = createFileInput(this.imageWrapper);
            fileInput.click();
        }

        this.image.style.cursor = 'pointer';
        this.api.tooltip.onHover(
            this.image,
            this.api.i18n.t('CTRL + Click to change image'),
            {
                placement: 'top',
            },
        );
        this.image.addEventListener('click', (e) => {
            if (e.ctrlKey) {
                var fileInput = createFileInput(this.imageWrapper);
                fileInput.click();
            }
        });

        this.imageWrapper.appendChild(this.image);

        return this.imageWrapper
    }
  
    save(blockContent) {
        const image = blockContent.querySelector('img');
        return {
            id: image.dataset.id,
            caption: image.dataset.caption,
            filePath: image.dataset.filePath,
        };
    }
}


class GoDjangoImagesTool {
    constructor({ data, api, config, block }) {
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
            title: 'Images',
            icon: GoDjangoImagesIcon,
        };
    }

    validate(savedData) {
        return savedData.filePath && savedData.filePath.length > 0 && savedData.id;
    }
  
    render(){
        this.imageWrapper = document.createElement('div');
        this.imageWrapper.classList.add('GoDjango-image-tool');

        this.image = document.createElement('img');
        this.image.classList.add(
            this.api.styles.block,
            "GoDjango-image-tool__image"
        );

        const throwError = (message) => {
            let element = document.createElement('p');
            element.textContent = message;
            element.style.color = 'red';
            this.imageWrapper.appendChild(element);
            return;
        }

        function createFileInput(wrapper) {
            var fileInput = document.createElement('input');
            fileInput.type = 'file';
            fileInput.accept = 'image/*';
            fileInput.style.display = 'none';
            wrapper.appendChild(fileInput);
            fileInput.addEventListener('change', async (e) => {

                let csrfTokenResponse = await fetch(`${this.config.uploadUrl}`, {
                    method: 'GET',
                })
                let csrfTokenData = await csrfTokenResponse.json();
                let csrftoken = csrfTokenData.csrf_token;

                var formData = new FormData();
                formData.append('file', fileInput.files[0]);
                formData.append('csrf_token', csrftoken);

                fetch(this.config.uploadUrl, {
                    method: 'POST',
                    body: formData,
                })
                .then(response => response.json())
                .then(data => {

                    if (data.status !== STATUS_SUCCESS) {
                        let message = this.api.i18n.t('Failed to upload image');
                        if (data.message) {
                            message = data.message;
                        }
                        throwError(message);
                        return;
                    }

                    this.data.id = data.id;
                    this.data.caption = data.caption;
                    this.data.filePath = data.filePath;
                    this.image.src = `${this.config.serveUrl}/${data.filePath}`
                    this.image.dataset.id = data.id;
                    this.image.dataset.filePath = data.filePath;
                    fileInput.remove();

                }).catch((error) => {
                    let message = this.api.i18n.t('Failed to upload image');
                    if (error.message) {
                        message = error.message;
                    }
                    throwError(message);
                })

            });
            return fileInput;
        }

        createFileInput = createFileInput.bind(this);
        if (this.data.filePath) {
            this.image.src = `${this.config.serveUrl}/${this.data.filePath}`
            this.image.dataset.id = this.data.id;
            this.image.dataset.caption = this.data.caption || '';
            this.image.dataset.filePath = this.data.filePath;
        } else {
            var fileInput = createFileInput(this.imageWrapper);
            fileInput.click();
        }

        this.image.style.cursor = 'pointer';
        this.api.tooltip.onHover(
            this.image,
            this.api.i18n.t('CTRL + Click to change image'),
            {
                placement: 'top',
            },
        );
        this.image.addEventListener('click', (e) => {
            if (e.ctrlKey) {
                var fileInput = createFileInput(this.imageWrapper);
                fileInput.click();
            }
        });

        this.imageWrapper.appendChild(this.image);

        return this.imageWrapper
    }
  
    save(blockContent) {
        const image = blockContent.querySelector('img');
        return {
            id: image.dataset.id,
            caption: image.dataset.caption,
            filePath: image.dataset.filePath,
        };
    }
}

window.GoDjangoImageTool = GoDjangoImageTool;
window.GoDjangoImagesTool = GoDjangoImagesTool;

