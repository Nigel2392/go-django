import { Controller, ActionEvent } from "@hotwired/stimulus";


class MenuController extends Controller<any> {
    declare subMenus: NodeListOf<HTMLElement>
    
    connect() {
        this.element.menuController = this;
        this.element.dataset.menuController = "true";
    }

    toggle(event?: ActionEvent) {
        if (this.element.classList.contains("open")) {
            this.close(event);
        } else {
            this.open(event);
        }
    }

    open(event?: ActionEvent) {
        this.element.classList.add("open");
        this.element.setAttribute("aria-expanded", "true");
    }

    close(event?: ActionEvent) {
        this.element.classList.remove("open");
        this.element.setAttribute("aria-expanded", "false");
        if (!this.subMenus) {
            this.subMenus = this.element.querySelectorAll("[data-menu-controller]")
        }
        console.log(this.subMenus);
        for (var i = 0; i < this.subMenus.length; i++) {
            var subMenu = this.subMenus[i] as any;
            subMenu.menuController.close();
        }
    }
}


export {
    MenuController,
};
