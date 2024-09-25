import { Controller } from "@hotwired/stimulus";
import Sortable from "sortablejs";

class SortableController extends Controller {
    declare sortable: Sortable
    declare itemsTarget: HTMLElement
    declare itemTargets: HTMLElement[]
    declare replaceValue: string
    static targets = ["item", "items"]
    static values = {
        replace: String,
    }

    connect() {



        this.sortable = Sortable.create(this.itemsTarget, {
            handle: ".list-block-field",
            onEnd: (event: Sortable.SortableEvent) => {
                this.reOrderItems();
            },
        } as any)
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