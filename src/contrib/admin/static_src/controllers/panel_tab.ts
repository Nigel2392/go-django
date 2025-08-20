import { Controller, ActionEvent } from "@hotwired/stimulus";

class TabPanelController extends Controller<HTMLElement> {
    static targets = [
      "tab",
    ];

    declare currentTab: HTMLElement;
    declare tabTargets: HTMLElement[];

    connect() {
        this.currentTab = this.tabTargets[0];
        this.currentTab.classList.add("active");

        if (window.location.hash) {
            const index = this.tabTargets.findIndex(tab => !!tab.querySelector(window.location.hash));
            if (index !== -1) {
                this.selectTab(index);
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

    selectTab(index: number) {
        this.tabTargets.forEach((tab, i) => {
            tab.classList.toggle("active", i === index);
        });
    }
}

export { TabPanelController };