import { Controller, ActionEvent } from "@hotwired/stimulus";
import slugify from "../utils/slugify";

class PanelController extends Controller<HTMLElement> {
    static values = {
        panel: { type: String },
    }
    static targets = [
      "heading",
      "linkIcon",
      "content",
    ];

    declare panelValue: string;

    declare hasLinkIconTarget: boolean;

    declare headingTarget: HTMLElement;
    declare linkIconTarget: HTMLElement;
    declare contentTarget: HTMLElement;

    get parentPanels() {
        var parentPanels = [];
        let parent = this.element.parentElement.closest(".panel");
        while (parent) {
            parentPanels.push(parent);
            parent = parent.parentElement.closest(".panel");
        }
        return parentPanels;
    }

    connect() {
        if (this.hasLinkIconTarget) {
            this.linkIconTarget.addEventListener(
                "click", this.scrollToContent.bind(this),
            );
        }

        if (this.element.classList.contains("collapsed")) {
            this.element.setAttribute("aria-expanded", "false");
        }

        if (this.panelValue) {
            setTimeout(() => {

                let hash = window.location.hash;
                if (hash === `#${this.panelValue}`) {
                    this.parentPanels.forEach(panel => {
                        panel.classList.remove("collapsed");
                        panel.setAttribute("aria-expanded", "true");
                    });

                    this.element.classList.remove("collapsed");
                    this.element.setAttribute("aria-expanded", "true");

                    this.scrollToContent();
                }

            }, 100);
        }
    }

    toggle(event: ActionEvent) {
        event.preventDefault();
        let collapsed = !this.element.classList.contains("collapsed");
        this.element.classList.toggle("collapsed", collapsed);
        this.element.setAttribute("aria-expanded", String(!collapsed));

    }

    scrollToContent() {
        this.contentTarget.scrollIntoView({
            behavior: "smooth",
            block: "start"
        });
    }
}

type TitlePanelControllerValues = {
    [key: string]: { type: any, default: any }
}

// panel which can bind to a slug input for pre-filling the result
class TitlePanelController extends Controller<HTMLElement> {
    static values: TitlePanelControllerValues = {
        outputids: { 
            type: Array<string>,
            default: [],
        }, 
    }

    declare outputidsValue: string[];

    connect() {
        super.connect();

        if (this.outputidsValue.length === 0) {
            console.error("No output IDs found for title panel controller");
            this.disconnect();
            return;
        }

        let outputs = this.outputidsValue.map(id => {
            let elem = document.getElementById(id) as any;
            if (elem && elem.tagName.toLowerCase() === "input" && elem.value === "") {
                elem.shouldSlugify = true;
            }

            elem.addEventListener("change", () => {
                if (elem.value === "") {
                    elem.shouldSlugify = true;
                } else {
                    elem.shouldSlugify = false;
                }
            });

            return elem;
        });
        if (outputs.length === 0) {
            console.error("No output found for panel title controller");
            this.disconnect();
            return;
        }

        let inputs = this.element.querySelectorAll("[data-panel-input-id]");
        if (inputs.length === 0) {
            console.error("No input found for panel title controller");
            this.disconnect();
            return;
        }

        if (inputs.length > 1) {
            console.error("Multiple inputs found for panel title controller, cannot bind");
            this.disconnect();
            return;
        }

        const input = inputs[0] as HTMLInputElement;
        input.addEventListener("input", this.updateOutput.bind(this));
    }

    updateOutput(event: Event) {
        const input = event.target as HTMLInputElement;
        const value = input.value;

        this.outputidsValue.forEach(id => {
            const output = document.getElementById(id) as any;
            if (!output || !output.shouldSlugify) {
                return;
            }

            if (output && output.tagName.toLowerCase() === "input") {
                output.value = slugify(value);
            }
        });
    }
}

export { PanelController, TitlePanelController };