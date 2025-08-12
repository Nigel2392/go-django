import { Controller, ActionEvent } from "@hotwired/stimulus";

class Tab {
    element: HTMLElement;
    index: number;
    tabs: TabController;

    constructor(element: HTMLElement, index: number, tabs: TabController) {
        this.element = element;
        this.index = index;
        this.tabs = tabs;
    }

    open() {
        this.tabs.open(this.index);
    }

    set content(content: string | HTMLElement) {
        if (typeof content === "string") {
            this.element.innerHTML = content;
        } else {
            this.element.innerHTML = "";
            this.element.appendChild(content);
        }
    }
}

type TabControllerOptions = {
    onTabOpen?: (tab: Tab) => void;
    onTabClose?: (tab: Tab) => void;
};

class TabController extends Controller<any> {
    activeTab: number;

    static targets = ["tab"];
    static values = {

    };

    declare readonly tabTargets: HTMLElement[];

    connect() {
        this.activeTab = 0;
    }

    disconnect() {

    }

    open(index: number) {
        this.openTab({ params: { tabIndex: index } });
    }

    getTab(index: number): Tab | undefined {
        const tabElement = this.tabTargets[index];
        if (tabElement) {
            return new Tab(tabElement, index, this);
        }
        return undefined;
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

