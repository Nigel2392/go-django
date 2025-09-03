import { Controller, ActionEvent } from "@hotwired/stimulus";
import { Chooser } from "../chooser/chooser";

type ChooserEvent = Event & { detail: { action: "open" | "close", modal: ChooserController }};

function newChooserEvent(action: "open" | "close", modal: ChooserController, event?: Event): ChooserEvent {
    return new CustomEvent("modal:" + action, {
        detail: {
            action: action,
            modal: modal,
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
        createurl: { type: String },
    };

    declare readonly titleValue:     string;
    declare readonly listurlValue:   string;
    declare readonly createurlValue: string;

    declare readonly previewTarget: HTMLDivElement;
    declare readonly inputTarget:   HTMLInputElement;

    connect() {
        this.element.chooserController = this;
        this.chooser = new Chooser({
            title:     this.titleValue,
            listurl:   this.listurlValue,
            createurl: this.createurlValue,
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
