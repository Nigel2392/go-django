import { Controller } from "@hotwired/stimulus";
import tippy from "tippy.js";
import 'tippy.js/dist/tippy.css';


class TippyController extends Controller<Element> {
    declare contentValue: string;
    declare placementValue: string;
    declare delayValue: number;
    declare durationValue: number;
    declare offsetValue: [number, number];
    static values = {
        content: String,
        placement: {
            type: String,
            default: "top",
        },
        delay: {
            type: Number,
            default: 0,
        },
        duration: {
            type: Number,
            default: 0,
        },
        offset: Array, 
    }
    declare tippyInstance: any;

    connect() {
        this.tippyInstance = tippy(this.element, {
            content: this.contentValue,
            placement: this.placementValue as any,
            delay: this.delayValue,
            duration: this.durationValue,
            offset: this.offsetValue,
        });
    }

    disconnect() {
        if (this.tippyInstance) {
            this.tippyInstance.destroy();
        }
    }
}

export {
    TippyController,
};
