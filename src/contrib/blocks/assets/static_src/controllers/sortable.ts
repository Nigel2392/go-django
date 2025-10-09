import Sortable from "sortablejs";

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
        handle: { type: String, default: ".list-block-field-drag-handle" },
        draggable: { type: String, default: ".list-block-field" },
    }

    private get sortableConfig() {
        return {
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
        )
    }

    itemsTargetConnected(elem: HTMLElement) {
        if (this.sortable) {
            this.sortable.destroy();
        }
        this.sortable = Sortable.create(
            elem, this.sortableConfig,
        )
    }

    replaceValues() {
        var replace = this.replaceValue;
        return replace.split(';');
    }

    reOrderItems() {
        var replace = this.replaceValues();
        for (var i = 0; i < this.itemTargets.length; i++) {
            var item = this.itemTargets[i];
            var replaceStr = item.dataset.replace;
            if (replaceStr) {
                replace = replaceStr.split(';');
            }

            for (var j = 0; j < replace.length; j++) {
                var value = replace[j];
                if (value.startsWith("[") && value.endsWith("]")) {
                    value = value.slice(1, -1);
                    if (value.startsWith("data-")) {
                        var attribute = value.slice(5);
                        item.dataset[attribute] = i.toString();
                    } else {
                        item.setAttribute(value, i.toString());
                    }
                } else {
                    var element = item.querySelector(value);
                    if (element) {
                        if (element instanceof HTMLInputElement) {
                            element.value = i.toString();
                        } else {
                            element.textContent = i.toString();
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