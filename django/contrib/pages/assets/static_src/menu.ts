import { Controller, ActionEvent } from "@hotwired/stimulus";

type PageNode = {
    id: number,
    title: string,
    path: string,
    depth: number,
    numchild: number,
    url_path: string,
    status_flags: number,
    page_id: number,
    content_type: string,
    created_at: string,
    updated_at: string,
}

type PageMenuResponse = {
    parent_item?: PageNode,
    items: PageNode[],
}

function buildTemplate(template: HTMLTemplateElement, vars: { [key: string]: string }) {
    let html = template.innerHTML;
    for (const key in vars) {
        html = html.replace(`__${key.toUpperCase()}__`, vars[key]);
    }
    const div = document.createElement("div");
    div.innerHTML = html;
    return div.firstElementChild as HTMLElement;
}

class PageMenuController extends Controller<HTMLElement> {
    declare submenuTarget: HTMLElement
    declare templateMenuHeaderTarget: HTMLTemplateElement
    declare templateHasSubpagesTarget: HTMLTemplateElement
    declare templateNoSubpagesTarget: HTMLTemplateElement
    declare urlValue: string
    static targets = [
        "submenu",
        "templateMenuHeader",
        "templateHasSubpages",
        "templateNoSubpages",
    ]
    static values = {
        url: String,
    }

    connect() {
        (this.element as any).menuController = this;
        this.element.dataset.menuController = "true";
        document.addEventListener("menu:open", (event: CustomEvent) => {
            if (event.detail.action === "open" && event.detail.menu !== this) {
                this.close();
            }
        })
    }

    toggle(event?: ActionEvent) {
        if (this.element.classList.contains("open")) {
            this.close(event);
        } else {
            this.open(event);
        }
    }

    private open(event?: ActionEvent) {
        this.element.classList.add("open");
        this.element.setAttribute("aria-expanded", "true");

        var openEvent = new CustomEvent("menu:open", {
            detail: {
                action: "open",
                menu: this
            }
        });
        document.dispatchEvent(openEvent);


        this.submenuTarget.innerHTML = "";

        fetch(this.urlValue)
            .then(response => response.json())
            .then(data => this.render(data));
    }

    private close(event?: ActionEvent) {
        this.element.classList.remove("open");
        this.element.setAttribute("aria-expanded", "false");
        this.submenuTarget.innerHTML = "";
    }

    private buildMenuItem(item: PageNode) {
        let template;
        if (item.numchild > 0) {
            template = this.templateHasSubpagesTarget;
        } else {
            template = this.templateNoSubpagesTarget;
        }
        return buildTemplate(template, {
            id: `page-${item.id}`,
            label: item.title,
            page_id: item.id.toString(),
        });
    }

    private fetchItems(menuItem: HTMLElement, params: { [key: string]: string } = null) {
        const pageId = menuItem.dataset.pageId;
        params = params || {};
        const query = new URLSearchParams(params);
        query.set("page_id", pageId);
        const url = this.urlValue + "?" + query.toString();
        fetch(url)
            .then(response => response.json())
            .then(data => {
                this.submenuTarget.innerHTML = "";
                this.render(data);
            });
    }

    private render(data: PageMenuResponse) {

        this.submenuTarget.innerHTML = "";

        if (data.parent_item) {
            const menuItem = buildTemplate(this.templateMenuHeaderTarget, {
                id: `page-${data.parent_item.id}`,
                label: data.parent_item.title,
                page_id: data.parent_item.id.toString(),
            });
            menuItem.addEventListener("click", (event) => {
                event.preventDefault();
                this.fetchItems(menuItem, {
                    get_parent: "true",
                });
            });
            this.submenuTarget.appendChild(menuItem);
        }

        for (const item of data.items) {
            const menuItem = this.buildMenuItem(item);
            menuItem.addEventListener("click", (event) => {
                event.preventDefault();
                this.fetchItems(menuItem);
            });
            this.submenuTarget.appendChild(menuItem);
        }
    }
}

export {
    PageMenuController,
};
