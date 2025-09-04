import { Controller } from "@hotwired/stimulus";

//  data-controller="tooltip" data-tooltip-content-value="%s" data-tooltip-placement-value="%s" data-tooltip-offset-value="[0, %v]

type sidebarMenuItem = HTMLElement & {
    textElement: HTMLElement | null;
    contentElement: HTMLElement | null;
};

export default class SidebarController extends Controller<HTMLElement> {
    private topLevelMenuItems: sidebarMenuItem[] = [];

    connect() {
        let topLevelMenuItems = Array.from(this.element.querySelectorAll(".sidebar-menu-item")) as sidebarMenuItem[];
        this.topLevelMenuItems = topLevelMenuItems.filter((item: sidebarMenuItem) => {
            item.textElement = item.querySelector(".menu-item-label[data-depth='0']") as HTMLElement;
            item.contentElement = item.querySelector(".menu-item-content[data-depth='0']") as HTMLElement;
            return item.textElement !== null && item.contentElement !== null;
        });

        this.checkTooltips();
    }

    toggle(event: Event) {
        event.preventDefault();
        let collapsed = this.isCollapsed();
        this.element.classList.toggle("collapsed", !collapsed);
        this.element.setAttribute("aria-expanded", String(!collapsed));


        this.checkTooltips(collapsed);
    }

    open() {
        if (this.isCollapsed()) {
            this.element.classList.remove("collapsed");
            this.element.setAttribute("aria-expanded", "true");
            this.checkTooltips(false);
        }
    }

    close() {
        if (!this.isCollapsed()) {
            this.element.classList.add("collapsed");
            this.element.setAttribute("aria-expanded", "false");
            this.checkTooltips(true);
        }
    }

    private isCollapsed(): boolean {
        return this.element.classList.contains("collapsed");
    }

    private checkTooltips(collapsed?: boolean) {
        if (collapsed === undefined) {
            collapsed = this.isCollapsed();
        }

        this.topLevelMenuItems.forEach((item) => {

            const tooltipController = this.application.getControllerForElementAndIdentifier(item.contentElement, "tooltip");
            if (collapsed) {
                if (!item.contentElement.getAttribute("data-controller")?.includes("tooltip")) {
                    item.contentElement.setAttribute("data-controller", (item.contentElement.getAttribute("data-controller") ?? "") + " tooltip");
                    item.contentElement.setAttribute("data-tooltip-content-value", item.textElement.textContent.trim());
                    item.contentElement.setAttribute("data-tooltip-placement-value", "right");
                    item.contentElement.setAttribute("data-tooltip-offset-value", "[0, 10]");
                }
            } else {
                if (item.contentElement.getAttribute("data-controller")?.includes("tooltip")) {
                    let attr = item.contentElement.getAttribute("data-controller");
                    attr = attr.replace("tooltip", "").trim();
                    if (attr === "") {
                        item.contentElement.removeAttribute("data-controller");
                    } else {
                        item.contentElement.setAttribute("data-controller", attr);
                    }
                    item.contentElement.removeAttribute("data-tooltip-content-value");
                    item.contentElement.removeAttribute("data-tooltip-placement-value");
                    item.contentElement.removeAttribute("data-tooltip-offset-value");
                    tooltipController?.disconnect();
                }
            }
        });
    }
}