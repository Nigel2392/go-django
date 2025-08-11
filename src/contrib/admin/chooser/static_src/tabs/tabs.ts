import { Controller, ActionEvent } from "@hotwired/stimulus";

class TabController extends Controller<any> {
    static targets = ["tab"];
    static values = {

    };

    declare readonly tabTargets: HTMLElement[];

    connect() {

    }

    disconnect() {

    }

    openTab({ params: { tabIndex } }: { params: { tabIndex: number } }) {
        var tabs = this.tabTargets;
        var activeTab: HTMLElement | undefined;
        tabs.forEach((tab, index) => {
            if (index === tabIndex) {
                tab.classList.add("active");
                activeTab = tab;
            } else {
                tab.classList.remove("active");
            }
        });
        return activeTab;
    }

    addTab(tab: HTMLElement) {
        this.tabTargets.push(tab);
    }
}

export {
    TabController,
};

