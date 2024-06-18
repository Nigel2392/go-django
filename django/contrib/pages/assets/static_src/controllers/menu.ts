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
    console.log(html)
    for (const key in vars) {
        html = html.replace(new RegExp(`__${key.toUpperCase()}__`, "g"), vars[key]);
    }
    console.log(html)
    const div = document.createElement("div");
    div.innerHTML = html;
    return div.firstElementChild as HTMLElement;
}

function newLoader(loadingText: string) {
    const loaderWrapper = document.createElement("div");
    loaderWrapper.classList.add("menu-loader-wrapper");
    loaderWrapper.textContent = loadingText;
    const loader = document.createElement("div");
    loader.classList.add("menu-loader");
    loaderWrapper.appendChild(loader);
    return loaderWrapper;

}

class PageMenuController extends Controller<HTMLElement> {
    declare submenuTarget: HTMLElement
    declare templateTarget: HTMLTemplateElement
    declare urlValue: string
    static targets = [
        "submenu",
        "template",
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

    levelUp(event: ActionEvent) {
        this.fetchItems(
            (event.currentTarget as HTMLElement).dataset.pageId, {
                get_parent: "true",
            }
        );
    }

    levelDown(event: ActionEvent) {
        this.fetchItems(
            (event.currentTarget as HTMLElement).dataset.pageId
        );
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


        this.fetchItems();
    }

    private close(event?: ActionEvent) {
        this.element.classList.remove("open");
        this.element.setAttribute("aria-expanded", "false");
        this.submenuTarget.innerHTML = "";
    }

    private fetchItems(menuItem: HTMLElement | number | string = "", params: { [key: string]: string } = null) {
        let pageId: string;
        if (menuItem instanceof HTMLElement) {
            pageId = menuItem.dataset.pageId;
        } else {
            pageId = menuItem.toString();
        }

        let loader = newLoader("Loading...");
        this.submenuTarget.appendChild(loader);

        params = params || {};
        const query = new URLSearchParams(params);
        query.set("page_id", pageId);
        const url = this.urlValue + "?" + query.toString();

        fetch(url)
        .then(response => response.json())
        .then(data => {
            this.submenuTarget.innerHTML = "";
            this.render(data);
        })
        .catch(error => {
            console.error("Error fetching menu items", error);
            this.submenuTarget.innerHTML = "";
            let errorContainer = document.createElement("div");
            errorContainer.classList.add("menu-load-error");
            errorContainer.textContent = `Error fetching menu items`;
            this.submenuTarget.appendChild(errorContainer);
        });
    }

    private render(data: PageMenuResponse) {
        if (data.parent_item) {
            const menuItem = buildTemplate(this.templateTarget, {
                id: `page-${data.parent_item.id}`,
                label: data.parent_item.title,
                page_id: data.parent_item.id.toString(),
            });
            menuItem.classList.add("header-menu-item");
            let levelDown = menuItem.querySelector(".level-down");
            if (levelDown) {
                levelDown.remove();
            }
            this.submenuTarget.appendChild(menuItem);
        }

        for (const item of data.items) {
            var menuItem = buildTemplate(this.templateTarget, {
                id: `page-${item.id}`,
                label: item.title,
                page_id: item.id.toString(),
            });

            let levelUp = menuItem.querySelector(".level-up");
            if (levelUp) {
                levelUp.remove();
            }

            if (item.numchild <= 0) {
                let levelDown = menuItem.querySelector(".level-down");
                if (levelDown) {
                    levelDown.remove();
                }
            }

            this.submenuTarget.appendChild(menuItem);
        }
    }
}

export {
    PageMenuController,
};
