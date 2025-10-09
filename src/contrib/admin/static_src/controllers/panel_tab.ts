import { Controller, ActionEvent } from "@hotwired/stimulus";

class TabPanelController extends Controller<HTMLElement> {
    static targets = [
      "tab", "control",
    ];

    declare hasParentTab: boolean;
    declare tabTargets: HTMLElement[];
    declare controlTargets: HTMLElement[];

    connect() {
        this.tabTargets[0].classList.add("active");
        this.controlTargets[0].classList.add("active");
        this.hasParentTab = this.element.parentElement.closest(`[data-controller="${this.identifier}"]`) !== null;

        if (!this.hasParentTab && window.location.hash && window.location.hash.startsWith("#tab-")) {
            const index = parseInt(window.location.hash.split("-").pop() || "0", 10);
            this.selectTab(index, false);
        }
    }

    select(event: ActionEvent) {
        event.preventDefault();

        if (event.params.index === undefined || event.params.index >= this.tabTargets.length) {
            console.warn("Invalid tab index:", event.params.index, this.tabTargets);
            return;
        }

        this.selectTab(parseInt(event.params.index, 10));
    }

    selectTab(index: number, setHash: boolean = true) {

        if (!this.hasParentTab && setHash) {
            window.location.hash = `#tab-${index}`;
        }

        this.tabTargets.forEach((tab, i) => {
            tab.classList.toggle("active", i === index);
        });
        this.controlTargets.forEach((control, i) => {
            control.classList.toggle("active", i === index);
        });
    }
}

export { TabPanelController };