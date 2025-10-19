import Sortable, { MultiDrag } from "sortablejs";

Sortable.mount(new MultiDrag());

class SortableController extends window.StimulusController {
    declare sortable: Sortable
    declare hasItemsTarget: boolean
    declare itemsTarget: HTMLElement
    declare itemTargets: HTMLElement[]
    declare replaceValue: string
    declare handleValue: string
    declare draggableValue: string
    static targets = ["item", "items"]
    static values = {
        replace: String,
        handle: { type: String, default: ".sequence-block-field-drag-handle" },
        draggable: { type: String, default: ".sequence-block-field" },
    }

    private get sortableConfig() {
        return {
            multiDrag: true,
            selectedClass: 'sort-selected',
            handle: this.handleValue,
            animation: 150,
            swapThreshold: 1,
            draggable: this.draggableValue,
            onEnd: (event: Sortable.SortableEvent) => {
                this.reOrderItems();
            },
        };
    }

    connect() {
        if (!this.hasItemsTarget) {
            console.error("SortableController: No items target found");
            return;
        }
        this.sortable = Sortable.create(
            this.itemsTarget, this.sortableConfig,
        );
    }

    itemsTargetConnected(elem: HTMLElement) {
        if (elem.dataset.sortable === "connected") {
            return;
        }
        if (this.sortable) {
            this.sortable.destroy();
        }
        elem.dataset.sortable = "connected";
        this.sortable = Sortable.create(
            elem, this.sortableConfig,
        )
    }

    moveItem(item: HTMLElement, direction: 'up' | 'down') {
        const items = this.itemTargets.filter(i => i.style.display !== 'none');
        const index = items.indexOf(item);
        if (index === -1) {
            console.error("SortableController: Item not found in items target");
            return;
        }

        let moved = true;
        if (direction === 'up' && index > 0) {
            const prevItem = items[index - 1];
            this.itemsTarget.insertBefore(item, prevItem);
        } else if (direction === 'down' && index < items.length - 1) {
            const nextItem = items[index + 1].nextSibling;
            this.itemsTarget.insertBefore(item, nextItem);
        } else {
            moved = false;
        }
        if (moved) {
            this.reOrderItems();
        }
    }

    itemTargetConnected(elem: HTMLElement) {
        this.reOrderItems();
    }

    replaceValues() {
        var replace = this.replaceValue;
        return replace.split(';');
    }

    reOrderItems() {
        var replace = this.replaceValues();
        var totalTargets = this.itemTargets.filter(item => item.style.display !== 'none');
        for (var i = 0; i < totalTargets.length; i++) {
            var item = totalTargets[i];
            var replaceStr = item.dataset.replace;
            if (replaceStr) {
                replace = replaceStr.split(';');
            }

            for (var j = 0; j < replace.length; j++) {
                var value = replace[j];
                var addOne = value.endsWith('+');
                let index = i;
                if (addOne) {
                    value = value.slice(0, -1);
                    index = i + 1;
                }
                if (value.startsWith("[") && value.endsWith("]")) {
                    value = value.slice(1, -1);
                    if (value.startsWith("data-")) {
                        var attribute = value.slice(5);
                        item.dataset[attribute] = index.toString();
                    } else {
                        item.setAttribute(value, index.toString());
                    }
                } else {
                    var element = item.querySelector(value);
                    if (element) {
                        if (element instanceof HTMLInputElement) {
                            element.value = index.toString();
                        } else {
                            element.textContent = index.toString();
                        }
                    } else {
                        console.error("Could not find element with selector", value);
                    }
                }
            }
        }
    }
}


export {
    SortableController,
};