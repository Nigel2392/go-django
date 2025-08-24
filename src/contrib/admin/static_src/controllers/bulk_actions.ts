import { Controller, ActionEvent } from "@hotwired/stimulus";

class BulkActionsController extends Controller<any> {

    static targets = ["execute", "checkbox", "selectAll"];

    declare hasSelectAllTarget: boolean;
    declare selectAllTarget: HTMLButtonElement;
    declare executeTargets: HTMLButtonElement[];
    declare checkboxTargets: HTMLInputElement[];

    connect() {
        this.changed(0);

        for (const checkbox of this.checkboxTargets) {
            checkbox.addEventListener("change", () => {
                this.changed();
            });
        }

        if (this.hasSelectAllTarget) {
            this.selectAllTarget.addEventListener("click", (e: Event) => {
                e.preventDefault();

                var amountChecked = this.checkboxTargets.filter(c => c.checked).length;
                if (amountChecked === this.checkboxTargets.length) {
                    this.checkboxTargets.forEach(c => {
                        c.checked = false;
                    });
                } else {
                    this.checkboxTargets.forEach(c => {
                        c.checked = true;
                    });
                }

                this.changed();
            });
        }
    }

    changed(amount?: number) {
        if (amount === undefined) {
            amount = this.checkboxTargets.filter(c => c.checked).length;
        }

        //  if (amount == this.checkboxTargets.length && this.hasSelectAllTarget) {
        //      this.selectAllTarget.disabled = true;
        //      this.selectAllTarget.classList.add("disabled");
        //  } else if (this.hasSelectAllTarget) {
        //      this.selectAllTarget.disabled = false;
        //      this.selectAllTarget.classList.remove("disabled");
        //  }

        this.executeTargets.forEach(btn => {
            btn.disabled = amount === 0;
            btn.classList.toggle("disabled", amount === 0);
        });
    }
    
}


export default BulkActionsController
