import { Controller, ActionEvent } from "@hotwired/stimulus";

class TabPanelController extends Controller<HTMLElement> {
    static targets = [
      "tab", "control",
    ];

    declare tabTargets: HTMLElement[];
    declare controlTargets: HTMLElement[];

    connect() {
        this.tabTargets[0].classList.add("active");
        this.controlTargets[0].classList.add("active");

        if (window.location.hash) {

            if (window.location.hash.startsWith("#tab-")) {
                const index = parseInt(window.location.hash.split("-").pop() || "0", 10);
                this.selectTab(index, false);
            } else {
                const index = this.tabTargets.findIndex(tab => !!tab.querySelector(window.location.hash));
                if (index !== -1) {
                    this.selectTab(index, false);
                }
            }
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

        if (setHash) {
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