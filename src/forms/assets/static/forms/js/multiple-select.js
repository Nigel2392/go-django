// multiple-select
// multiple-select-controls
// multiple-select-remove
// multiple-select-remove-all
// multiple-select-add-all
// multiple-select-add
// multiple-select-chooser
// data-deselected
// data-selected

class MultipleSelect {
    constructor(options) {
        const {
            selector = '.multiple-select',
            selectedBox = '[data-selected]',
            deselectedBox = '[data-deselected]',
        } = options;

        this.wrapper = document.querySelector(selector);
        this.selected = this.wrapper.querySelector(selectedBox);
        this.deselected = this.wrapper.querySelector(deselectedBox);
        this.controls = this.wrapper.querySelector(`${selector}-controls`);

        this.addSelected = this.controls.querySelector(`${selector}-add`);
        this.removeSelected = this.controls.querySelector(`${selector}-remove`);
        this.addAllSelected = this.controls.querySelector(`${selector}-add-all`);
        this.removeAllSelected = this.controls.querySelector(`${selector}-remove-all`);

        this.init();
    }

    init() {
        this.addSelected.addEventListener('click', this.addSelectedListener.bind(this));
        this.removeSelected.addEventListener('click', this.removeSelectedListener.bind(this));
        this.addAllSelected.addEventListener('click', this.addAllSelectedListener.bind(this));
        this.removeAllSelected.addEventListener('click', this.removeAllSelectedListener.bind(this));
        this.deselected.addEventListener('click', this.deselectedListener.bind(this));
        this.selected.addEventListener('click', this.selectedListener.bind(this));
    }

    addSelectedListener() {
        const selected = this.deselected.querySelectorAll('option:checked');
        selected.forEach((option) => {
            this.selected.appendChild(option);
            option.selected = true;
        });
    }

    removeSelectedListener() {
        const selected = this.selected.querySelectorAll('option:checked');
        selected.forEach((option) => {
            this.deselected.appendChild(option);
            option.selected = false;
        });
    }

    addAllSelectedListener() {
        const options = this.deselected.querySelectorAll('option');
        options.forEach((option) => {
            this.selected.appendChild(option);
            option.selected = true;
        });
    }

    removeAllSelectedListener() {
        const options = this.selected.querySelectorAll('option');
        options.forEach((option) => {
            this.deselected.appendChild(option);
            option.selected = false;
        });
    }

    deselectedListener(event) {
        event.preventDefault();
        const option = event.target;
        if (option.tagName === 'OPTION') {
            this.selected.appendChild(option);
            option.selected = true;
        }
    }

    selectedListener(event) {
        const option = event.target;
        if (option.tagName === 'OPTION') {
            this.deselected.appendChild(option);
            option.selected = false;
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const multipleSelect = new MultipleSelect({
        selector: '.multiple-select',
        selectedBox: '[data-selected]',
        deselectedBox: '[data-deselected]',
    });
});