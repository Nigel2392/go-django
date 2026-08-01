import { Controller } from "@hotwired/stimulus";

//  data-controller="tooltip" data-tooltip-content-value="%s" data-tooltip-placement-value="%s" data-tooltip-offset-value="[0, %v]

type sidebarMenuItem = HTMLElement & {
    textElement: HTMLElement | null;
    contentElement: HTMLElement | null;
};

export default class SidebarController extends Controller<HTMLElement> {
    static values = {
        cookieName: {
            type: String,
            default: "sidebar_collapsed",
        },
    }
    private topLevelMenuItems: sidebarMenuItem[] = [];
    declare cookieNameValue: string;


    connect() {
        let topLevelMenuItems = Array.from(this.element.querySelectorAll(".sidebar-menu-item")) as sidebarMenuItem[];
        this.topLevelMenuItems = topLevelMenuItems.filter((item: sidebarMenuItem) => {
            item.textElement = item.querySelector(".menu-item-label[data-depth='0']") as HTMLElement;
            item.contentElement = item.querySelector(".menu-item-content[data-depth='0']") as HTMLElement;
            return item.textElement !== null && item.contentElement !== null;
        });

        var isCollapsed = this.isCollapsed();
        var cookie = window.getCookie(this.cookieNameValue);
        if (cookie === "true" && isCollapsed) {
            this.open();
        } else if (cookie === "false" && !isCollapsed) {
            this.close();
        } else {
            this.checkTooltips();
        }
    }

    toggle(event: Event) {
        event.preventDefault();
        
        let collapsed = this.isCollapsed();
        if (collapsed) {
            this.open(true);
        } else {
            this.close(true);
        }
    }

    open(exec?: boolean) {
        if (exec || this.isCollapsed()) {
            this.element.classList.remove("collapsed");
            this.element.setAttribute("aria-expanded", "true");
            this.checkTooltips();
        }

        window.setCookie(this.cookieNameValue, "true", 365);
    }

    close(exec?: boolean) {
        if (exec || !this.isCollapsed()) {
            this.element.classList.add("collapsed");
            this.element.setAttribute("aria-expanded", "false");
            this.checkTooltips();
        }

        window.setCookie(this.cookieNameValue, "false", 365);
    }

    private isCollapsed(): boolean {
        return this.element.classList.contains("collapsed");
    }

    private checkTooltips() {
        const collapsed = this.isCollapsed();
        this.topLevelMenuItems.forEach((item) => {

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
                }
            }
        });
    }
}