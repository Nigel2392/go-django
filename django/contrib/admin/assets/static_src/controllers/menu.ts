import { Controller, ActionEvent } from "@hotwired/stimulus";



const OPEN = "open"
const CLOSE = "close"

type MenuOpenCloseEvent = Event & { detail: { action: typeof OPEN | typeof CLOSE, menu: MenuController }};

class MenuController extends Controller<any> {
    declare subMenus: NodeListOf<HTMLElement>

    static addEventListener(func: (event: MenuOpenCloseEvent) => void) {
        document.addEventListener("menu:open", func);
    }

    listenForClose(event: MenuOpenCloseEvent) {
        if (event.detail.action === OPEN && event.detail.menu !== this) {
            if (!this.element.contains(event.detail.menu.element) && !event.detail.menu.element.contains(this.element)) {
                this.close();
            }
        }
    }
    
    connect() {
        this.element.menuController = this;
        this.element.dataset.menuController = "true";
        (this.constructor as any).addEventListener(this.listenForClose.bind(this));
    }

    contains(element: HTMLElement | MenuController | Controller) {
        if (element instanceof MenuController || element instanceof Controller) {
            element = element.element;
        }
        return this.element.contains(element);
    }

    toggle(event?: ActionEvent) {
        if (this.element.classList.contains("open")) {
            this.close(event);
        } else {
            this.open(event);
            var openEvent = new CustomEvent("menu:open", {
                detail: {
                    action: OPEN,
                    menu: this
                }
            });
            document.dispatchEvent(openEvent);
        }
    }

    private open(event?: ActionEvent) {
        this.element.classList.add("open");
        this.element.setAttribute("aria-expanded", "true");
    }

    private close(event?: ActionEvent) {
        this.element.classList.remove("open");
        this.element.setAttribute("aria-expanded", "false");
        if (!this.subMenus) {
            this.subMenus = this.element.querySelectorAll("[data-menu-controller]")
        }
        for (var i = 0; i < this.subMenus.length; i++) {
            var subMenu = this.subMenus[i] as any;
            subMenu.menuController.close();
        }
    }
}


export {
    MenuController,
};
