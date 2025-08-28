// multiple-select
// multiple-select-controls
// multiple-select-remove
// multiple-select-remove-all
// multiple-select-add-all
// multiple-select-add
// multiple-select-chooser
// data-deselected
// data-selected

import { Controller } from "@hotwired/stimulus";

class MultipleSelectController extends Controller<any> {
    declare addTarget: HTMLElement;
    declare removeTarget: HTMLElement;
    declare addAllTarget: HTMLElement;
    declare removeAllTarget: HTMLElement;
    declare selectedTarget: HTMLElement;
    declare deselectedTarget: HTMLElement;

    static targets = [
        "add",
        "remove",
        "addAll",
        "removeAll",
        "selected",
        "deselected"
    ];

    connect() {
        this.addTarget.addEventListener('click', this.addSelectedListener.bind(this));
        this.removeTarget.addEventListener('click', this.removeSelectedListener.bind(this));
        this.addAllTarget.addEventListener('click', this.addAllSelectedListener.bind(this));
        this.removeAllTarget.addEventListener('click', this.removeAllSelectedListener.bind(this));
        this.selectedTarget.addEventListener('dblclick', this.selectedListener.bind(this));
        this.deselectedTarget.addEventListener('dblclick', this.deselectedListener.bind(this));
        this.selectedTarget.querySelectorAll('option').forEach((option) => option.selected = true);
    }

    addSelectedListener() {
        (this.deselectedTarget.querySelectorAll('option:checked') as NodeListOf<HTMLOptionElement>).forEach((option) => {
            this.selectedTarget.appendChild(option);
            option.selected = true;
        });
        this.selectedTarget.querySelectorAll('option').forEach((option) => option.selected = true);
    }

    removeSelectedListener() {
        (this.selectedTarget.querySelectorAll('option:checked') as NodeListOf<HTMLOptionElement>).forEach((option) => {
            this.deselectedTarget.appendChild(option);
            option.selected = false;
        });
        this.selectedTarget.querySelectorAll('option').forEach((option) => option.selected = true);
    }

    addAllSelectedListener() {
        this.deselectedTarget.querySelectorAll('option').forEach((option) => {
            this.selectedTarget.appendChild(option);
            option.selected = true;
        });
    }

    removeAllSelectedListener() {
        this.selectedTarget.querySelectorAll('option').forEach((option) => {
            this.deselectedTarget.appendChild(option);
            option.selected = false;
        });
    }

    deselectedListener(event: Event) {
        event.preventDefault();
        const option = event.target as HTMLOptionElement;
        if (option.tagName === 'OPTION') {
            this.selectedTarget.appendChild(option);
            option.selected = true;
        }
        this.selectedTarget.querySelectorAll('option').forEach((option) => option.selected = true);
    }

    selectedListener(event: Event) {
        event.preventDefault();
        const option = event.target as HTMLOptionElement;
        if (option.tagName === 'OPTION') {
            this.deselectedTarget.appendChild(option);
            option.selected = false;
        }
        this.selectedTarget.querySelectorAll('option').forEach((option) => option.selected = true);
    }
}

export {
    MultipleSelectController
};

