import { Controller, ActionEvent } from "@hotwired/stimulus";

class SidePanelsController extends Controller<any> {

    static targets = ["control", "panel"];

    declare controlTargets: HTMLElement[];
    declare panelTargets: HTMLElement[];

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
}

export default SidePanelsController;
