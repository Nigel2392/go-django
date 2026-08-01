import { Controller, ActionEvent } from "@hotwired/stimulus";
import slugify from "../utils/slugify";
import { PanelElement } from "./panel";
import { openAnimator } from "../utils/animator";
import flash, { FlashOptions } from "../utils/flash";

class ManagementFormElement {
    element: HTMLElement;
    prefix: string;
    TotalFormsInput!: HTMLInputElement;
    InitialFormsInput!: HTMLInputElement;
    MinNumFormsInput!: HTMLInputElement;
    MaxNumFormsInput!: HTMLInputElement;

    constructor(prefix: string, element: HTMLElement) {
        this.element = element;
        this.prefix = prefix;
        this.TotalFormsInput = this.element.querySelector(`input[name="${this.prefixName("TOTAL_FORMS")}"]`) as HTMLInputElement;
        this.InitialFormsInput = this.element.querySelector(`input[name="${this.prefixName("INITIAL_FORMS")}"]`) as HTMLInputElement;
        this.MinNumFormsInput = this.element.querySelector(`input[name="${this.prefixName("MIN_NUM_FORMS")}"]`) as HTMLInputElement;
        this.MaxNumFormsInput = this.element.querySelector(`input[name="${this.prefixName("MAX_NUM_FORMS")}"]`) as HTMLInputElement;
    }

    private prefixName(name: string): string {
        return this.prefix.replace(/__FIELD__/g, name);
    }

    get totalForms(): number {
        return parseInt(this.TotalFormsInput.value, 10);
    }
    set totalForms(value: number) {
        this.TotalFormsInput.value = value.toString();
    }
    get initialForms(): number {
        return parseInt(this.InitialFormsInput.value, 10);
    }
    get minNumForms(): number {
        return parseInt(this.MinNumFormsInput.value, 10);
    }
    get maxNumForms(): number {
        return parseInt(this.MaxNumFormsInput.value, 10);
    }
}

class InlinePanelController extends Controller<PanelElement> {
    static values = {
        prefix: String,
        mgmtPrefix: String,
        minForms: { type: Number, default: 0 },
        maxForms: { type: Number, default: Infinity },
    }
    static targets = [
        "template",
        "management",
        "forms",
    ];

    declare prefixValue: string;
    declare mgmtPrefixValue: string;
    declare minFormsValue: number;
    declare maxFormsValue: number;
    declare hasTemplateTarget: boolean;
    declare managementTarget: HTMLElement;
    declare templateTarget: HTMLElement;
    declare formsTarget: HTMLElement;

    declare managementForm: ManagementFormElement;
    declare lastFormIndex: number;
    declare totalForms: number; // different from managementForm.totalForms which includes deleted forms
    
    connect() {
        this.lastFormIndex = this.formsTarget.children.length - 1;
        if (this.lastFormIndex < 0) {
            this.lastFormIndex = 0;
        }

        this.managementForm = new ManagementFormElement(this.mgmtPrefixValue, this.managementTarget);
        this.totalForms = this.managementForm.totalForms;
    }

    addFormAction(event: ActionEvent) {
        event.preventDefault();
        let specifier = null;
        if ("id" in event.params) {
            specifier = event.params.id;
        } else if ("index" in event.params) {
            specifier = parseInt(event.params.index, 10);
        }
        let where = "end";
        if ("where" in event.params) {
            where = event.params.where;
        }
        this.addForm(specifier, where as "start" | "end");
    }

    removeFormAction(event: ActionEvent) {
        event.preventDefault();
        this.removeForm(event.params.id, event.params.name);
    }

    addForm(specifier: string | number | null, where: "start" | "end" = "end") {
        if (!this.hasTemplateTarget) {
            console.error("No template target found for inline panel, cannot add form.");
            return;
        }

        if (this.totalForms >= this.maxFormsValue) {
            console.warn("Maximum number of forms reached, cannot add more.");
            this.flash(this.formsTarget);
            return;
        }

        // get the index of the targeted element
        let newFormHtml = this.templateTarget.innerHTML.
            replace(/__PREFIX__/g, this.prefixValue).
            replace(/__INDEX__/g, (this.lastFormIndex + 1).toString()).
            replace(/__SLUGIFY\(([a-zA-Z0-9_-]*?)\)__/g, (match, p1) => {
                return slugify(p1);
            });

        let newFormElem = document.createElement("div");
        newFormElem.innerHTML = newFormHtml;

        const animator = new openAnimator({
            duration: 300,
            onAdded: (elem) => {
                elem.style.transition = "opacity 300ms ease";
            },
            onFinished: (elem) => {
                elem.style.transition = "";
                if (!elem.style) {
                    elem.removeAttribute("style");
                }
            },
            animFrom: { opacity: "0" },
            animTo: { opacity: "1" },
        });

        let targetedElement = this.getTargetElement(specifier);
        if (where === "start" && !targetedElement) {
            targetedElement = this.formsTarget.firstChild as PanelElement;
        }

        if (newFormElem.children.length > 1) {
            console.error("Template for inline panel contains multiple root elements, cannot add form.");
            return;
        }

        var panelElem = newFormElem.children[0] as PanelElement;
        if (!panelElem || !panelElem.classList.contains("panel")) {
            console.error("Template for inline panel does not contain a root element with class 'panel', cannot add form.");
            return;
        }

        animator.addElement(panelElem);

        switch (where) {
        case "start":
            this.formsTarget.insertBefore(
                panelElem,
                targetedElement,
            );
            break;
        case "end":
            const targetElementIndex = targetedElement ? Array.from(this.formsTarget.children).indexOf(targetedElement) : -1;
            if (targetElementIndex !== -1 && targetElementIndex < this.formsTarget.children.length - 1) {
                this.formsTarget.insertBefore(panelElem, targetedElement.nextSibling);
            } else {
                this.formsTarget.appendChild(panelElem);
            }
            break;
        }

        animator.start();
        
        this.flash(panelElem, {
            color: 'green',
            duration: 300,
            iters: 1,
            delay: 20,
        });

        this.lastFormIndex += 1;
        this.managementForm.totalForms += 1;
        this.totalForms += 1;
    }

    removeForm(specifier: string | number, nameFmt: string) {
        const targetedElement = this.getTargetElement(specifier);
        if (!targetedElement) {
            console.error("No targeted element found for inline panel, cannot remove form.");
            return;
        }

        if (this.totalForms <= this.minFormsValue) {
            console.warn("Minimum number of forms reached, cannot remove more.");
            this.flash(targetedElement.panelBody.firstElementChild);
            return;
        }

        let name = nameFmt.replace(/__FIELD__/g, "__DELETE__");
        let selector = `input[name="${name}"]`;
        let deletedInput = targetedElement.querySelector(selector) as HTMLInputElement;
        if (!deletedInput) {
            console.error(`No deleted input found for inline panel ${specifier} with selector ${selector}, cannot remove form.`);
            return;
        }

        if (deletedInput.value === "true" || deletedInput.value === "on" || deletedInput.value === "1") {
            console.warn(`Form ${specifier} already marked for deletion, cannot remove again.`);
            this.flash(targetedElement.panelBody.firstElementChild);
            return;
        }

        this.flash(this.formsTarget, {
            color: 'orange',
            duration: 300,
            iters: 1,
            delay: 20,
        });

        deletedInput.value = "true";
        targetedElement.style.display = "none";
        this.totalForms -= 1;
        return;
    }

    private flash(element: Element | null, opts: FlashOptions = { color: 'red', duration: 200, iters: 2, delay: 0 }) {
        flash(element, opts);
    }

    private getTargetElement(specifier: string | number | null): PanelElement | null {
        let targetedElement: PanelElement | null = null;
        let typeOfSpecifier = typeof specifier;
        if (specifier === null || typeOfSpecifier === "undefined" || (typeOfSpecifier === "string" && specifier === "")) {
            targetedElement = this.formsTarget.lastElementChild as PanelElement;
        } else if (typeOfSpecifier === "string") {
            targetedElement = document.getElementById(specifier as string) as PanelElement;
        } else if (typeOfSpecifier === "number") {
            if (this.formsTarget.children.length > (specifier as number)) {
                targetedElement = this.formsTarget.children[specifier as number] as PanelElement;
            } else {
                targetedElement = this.formsTarget.lastElementChild as PanelElement;
                console.warn(`Index ${specifier} out of bounds, appending to end instead.`);
            }
        }
        return targetedElement;
    }
}

export { InlinePanelController };