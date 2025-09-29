import { Controller, ActionEvent } from "@hotwired/stimulus";
import slugify from "../utils/slugify";
import { PanelElement } from "./panel";

type openAnimatorOptions = {
    elements?: HTMLElement | HTMLElement[];
    duration?: number;
    easing?: string;
    animFrom?: { [key: string]: string };
    animTo?: { [key: string]: string };
    onAdded?: (elem: HTMLElement) => void;
    onStart?: (elem: HTMLElement) => void;
    onFinished?: (elem: HTMLElement) => void;
}

class openAnimator {
    elements: HTMLElement[];
    options: openAnimatorOptions;

    constructor(options: openAnimatorOptions = { duration: 300, easing: "ease" }) {
        this.options = options;
    
        if (options.elements) {
            if (options.elements instanceof HTMLElement) {
                this.elements = [options.elements];
            } else {
                this.elements = options.elements;
            }
        } else {
            this.elements = [];
        }
    }

    addElement(...elements: HTMLElement[]) {
        elements.forEach(elem => {
            this.options.onAdded?.(elem);
        });
        this.elements.push(...elements);
    }

    start() {
        this.elements.forEach(elem => {
            this.options.onStart?.(elem);

            // Start the animation on the next frame so start/end are distinct
            requestAnimationFrame(() => {
                const anim = elem.animate(
                    [
                        { height: '0px', ...this.options.animFrom },
                        { height: `${elem.offsetHeight}px`, ...this.options.animTo },
                    ],
                    {
                        duration: this.options.duration || 300,
                        easing: this.options.easing || 'ease',
                        fill: 'none', // don't retain the final numeric height
                    }
                );
              
                anim.onfinish = () => {
                    this.options.onFinished?.(elem);
                };
            });
        });
    }
}


class InlinePanelController extends Controller<HTMLElement> {
    static values = {
        prefix: String,
        minForms: { type: Number, default: 0 },
        maxForms: { type: Number, default: Infinity },
    }
    static targets = [
      "template",
      "forms",
    ];

    declare prefixValue: string;
    declare minFormsValue: number;
    declare maxFormsValue: number;
    declare hasTemplateTarget: boolean;
    declare templateTarget: HTMLElement;
    declare formsTarget: HTMLElement;
    
    declare lastFormIndex: number;
    declare formCount: number;
    
    connect() {
        this.lastFormIndex = this.formsTarget.children.length - 1;
        if (this.lastFormIndex < 0) {
            this.lastFormIndex = 0;
        }

        this.formCount = this.formsTarget.children.length;
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

        if (this.formCount >= this.maxFormsValue) {
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

        for (let i = 0; i < newFormElem.children.length; i++) {
            animator.addElement(newFormElem.children[i] as HTMLElement);

            switch (where) {
            case "start":
                this.formsTarget.insertBefore(
                    newFormElem.children[i] as HTMLElement,
                    targetedElement,
                );
                targetedElement = newFormElem.children[i] as PanelElement;
                break;
            case "end":
                const targetElementIndex = targetedElement ? Array.from(this.formsTarget.children).indexOf(targetedElement) : -1;
                if (targetElementIndex !== -1 && targetElementIndex < this.formsTarget.children.length - 1) {
                    this.formsTarget.insertBefore(
                        newFormElem.children[i] as HTMLElement,
                        targetedElement.nextSibling,
                    );
                    targetedElement = newFormElem.children[i] as PanelElement;
                } else {
                    this.formsTarget.appendChild(newFormElem.children[i] as HTMLElement);
                    targetedElement = newFormElem.children[i] as PanelElement;
                }
                break;
            }

            this.formCount++;
        }
        
        animator.start();

        this.lastFormIndex++;
    }

    removeForm(specifier: string | number, nameFmt: string) {
        const targetedElement = this.getTargetElement(specifier);
        if (!targetedElement) {
            console.error("No targeted element found for inline panel, cannot remove form.");
            return;
        }

        if (this.formCount <= this.minFormsValue) {
            console.warn("Minimum number of forms reached, cannot remove more.");
            this.flash(targetedElement.panelBody.firstElementChild);
            return;
        }

        let name = nameFmt.replace(/__FIELD__/g, "__DELETED__");
        let deletedInput = targetedElement.querySelector(`input[name="${name}"]`) as HTMLInputElement;
        if (!deletedInput || deletedInput.value === "true") {
            console.error("No deleted input found for inline panel, cannot remove form.");
            return;
        }

        deletedInput.value = "true";
        targetedElement.style.display = "none";
        this.formCount--;
        return;
    }

    private flash(element: Element | null) {
        if (!element) return;
        element.animate(
            [
                { boxShadow: '0 0 0px red' },
                { boxShadow: '0 0 10px red' },
                { boxShadow: '0 0 0px red' },
            ],
            {
                duration: 200,
                easing: "linear",
                fill: 'none', // don't retain the final numeric height
                iterations: 2,
                direction: 'alternate',
            }
        );
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