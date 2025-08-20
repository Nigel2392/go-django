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

export { PanelController };