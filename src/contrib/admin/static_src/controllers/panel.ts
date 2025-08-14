import { Controller, ActionEvent } from "@hotwired/stimulus";

class PanelController extends Controller<any> {
    static targets = [
      "heading",
      "icon",
      "content",
    ];
    
    declare headingTarget: HTMLElement;
    declare iconTarget: HTMLElement;
    declare contentTarget: HTMLElement;

    connect() {
        console.log("PanelController connected");
    }

    toggle(event: ActionEvent) {
        event.preventDefault();
        this.element.classList.toggle("collapsed");
    }
}

export { PanelController };