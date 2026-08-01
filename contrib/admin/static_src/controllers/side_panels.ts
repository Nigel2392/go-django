import { Controller, ActionEvent } from "@hotwired/stimulus";

class SidePanelsController extends Controller<any> {

    static targets = ["control", "panel", "panels"];

    static values = {
        defaultPanelId: String
    }

    declare controlTargets: HTMLElement[];
    declare panelTargets: HTMLElement[];
    declare panelsTarget: HTMLElement;
    declare defaultPanelIdValue: string;
    declare hasDefaultPanelIdValue: boolean;

    connect() {
        this.panelTargets.forEach(p => {
            p.classList.remove("active");
        });

        if (this.hasDefaultPanelIdValue) {
            this.openPanel(this.defaultPanelIdValue);
        }
    }

    openPanel(panelId: string) {
        this.panelTargets.forEach(p => {
            if (p.id === panelId) {
                if (p.classList.contains("active")) {
                    this.panelsTarget.classList.remove("fullscreen");
                    p.classList.remove("active");
                    return;
                }
                p.classList.add("active");
            } else {
                p.classList.remove("active");
            }
        });
    }

    open(event: ActionEvent) {
        let panelId = event.params.id;
        this.openPanel(panelId);
    }

    close(event: ActionEvent) {
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
