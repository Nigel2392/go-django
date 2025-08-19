import { Controller, ActionEvent } from "@hotwired/stimulus";

class PanelController extends Controller<HTMLElement> {
    static values = {
        panel: { type: String }
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

        let hash = window.location.hash;
        if (hash === `#${this.panelValue}`) {
            this.parentPanels.forEach(panel => panel.classList.remove("collapsed"));
            this.element.classList.remove("collapsed");
            this.scrollToContent();
        }
    }

    toggle(event: ActionEvent) {
        event.preventDefault();
        this.element.classList.toggle("collapsed");
    }

    scrollToContent() {
        this.contentTarget.scrollIntoView({ behavior: "smooth" });
    }
}

export { PanelController };