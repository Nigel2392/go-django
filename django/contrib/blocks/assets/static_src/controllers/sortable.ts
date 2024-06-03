import { Controller } from "@hotwired/stimulus";
import Sortable from "sortablejs";

class SortableController extends Controller {
    declare sortable: Sortable
    declare itemsTarget: HTMLElement
    declare itemTargets: HTMLElement[]
    declare replaceValue: string
    declare replaceKeyValue: string

    static targets = ["item", "items"]
    static values = {
        replace: String,
        replaceKey: String,
    }

    connect() {
        const captureGroupKey = this.replaceKeyValue
        const captureGroupValue = this.replaceValue
        const regex = new RegExp(`${captureGroupValue}`, "g")

        this.sortable = Sortable.create(this.itemsTarget, {
            handle: ".list-block-field",
            onEnd: (event: Sortable.SortableEvent) => {

            },
        } as any)
    }
}


export {
    SortableController,
};