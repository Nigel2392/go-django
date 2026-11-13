import { Controller, ActionEvent } from "@hotwired/stimulus";
import { Chooser } from "./chooser/chooser";

type ChooserEvent = Event & { detail: { action: "open" | "close" | "select", modal: ChooserController, data?: any }};

function newChooserEvent(action: "open" | "close" | "select", modal: ChooserController, event?: Event, data?: any): ChooserEvent {
    return new CustomEvent("modal:" + action, {
        detail: {
            action: action,
            modal: modal,
            data: data,
            originalEvent: event,
        }
    }) as ChooserEvent;
}

class ChooserController extends Controller<any> {
    chooser: Chooser;

    static targets = ["preview", "input"];
    static values = {
        title:     { type: String },
        listurl:   { type: String },
    };

    declare readonly titleValue:     string;
    declare readonly listurlValue:   string;

    declare readonly previewTarget: HTMLDivElement;
    declare readonly inputTarget:   HTMLInputElement;

    connect() {
        this.element.chooserController = this;
        this.chooser = new Chooser({
            title:     this.titleValue,
            listurl:   this.listurlValue,
            onChosen:  this.select.bind(this),
        });
    }

    disconnect() {
        this.element.chooserController = null;
        this.element.dataset.chooserController = "false";
        this.chooser.disconnect();
    }

    select(value: string, previewText: string) {
        this.inputTarget.value = value;
        this.previewTarget.innerHTML = previewText;
        this.element.dispatchEvent(newChooserEvent(
            "select", this, null, { value: value, previewText: previewText }
        ))
    }


    async open(event?: ActionEvent) {
        await this.chooser.open();
        await this.element.dispatchEvent(newChooserEvent("open", this, event));
    }

    async close(event?: ActionEvent) {
        await this.chooser.close();
        await this.element.dispatchEvent(newChooserEvent("close", this, event));
    }

    async clear(event?: ActionEvent) {
        if (this.inputTarget.value === "") {
            return;
        }
        
        this.inputTarget.value = "";
        this.previewTarget.innerHTML = "";
        
        const currentColor = getComputedStyle(this.element).borderColor;
        this.element.animate([{ borderColor: "red" }, { borderColor: currentColor }], {
            fill: "forwards",
            duration: 300,
            easing: "ease-out",
        });
    }
}

export {
    ChooserEvent,
    ChooserController,
};
