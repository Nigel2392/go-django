import { Controller, ActionEvent } from "@hotwired/stimulus";

class SidePanelsController extends Controller<any> {

    static targets = ["control", "panel", "panels"];

    declare controlTargets: HTMLElement[];
    declare panelTargets: HTMLElement[];
    declare panelsTarget: HTMLElement;

    connect() {
    }

    open(event: ActionEvent) {
        let panelId = event.params.id;
        this.panelTargets.forEach(p => {
            if (p.id === panelId) {
                if (p.classList.contains("active")) {
                    p.classList.remove("active");
                    return;
                }
                p.classList.add("active");
            } else {
                p.classList.remove("active");
            }
        });
    }

    close(event: ActionEvent) {
        let panelId = event.params.id;
        this.panelTargets.forEach(p => {
            p.classList.remove("active");
        });
        this.panelsTarget.classList.remove("fullscreen");
    }

    fullscreen(event: ActionEvent) {
        this.panelsTarget.classList.toggle("fullscreen");
    }
}

export default SidePanelsController;
