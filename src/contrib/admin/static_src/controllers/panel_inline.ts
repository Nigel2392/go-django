import { Controller, ActionEvent } from "@hotwired/stimulus";
import slugify from "../utils/slugify";

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
    }
    static targets = [
      "template",
      "forms",
    ];

    declare prefixValue: string;
    declare hasTemplateTarget: boolean;
    declare templateTarget: HTMLElement;
    declare formsTarget: HTMLElement;
    declare lastFormIndex: number;
    
    connect() {
        this.lastFormIndex = this.formsTarget.children.length - 1;
        if (this.lastFormIndex < 0) {
            this.lastFormIndex = 0;
        }
    }

    addFormAction(event: ActionEvent) {
        event.preventDefault();
        const id = event.params.id;
        this.addForm(id);
    }

    removeFormAction(event: ActionEvent) {
        event.preventDefault();
        const id = event.params.id;
        this.removeForm(id);
    }

    addForm(specifier: string | number | null) {
        if (!this.hasTemplateTarget) {
            console.error("No template target found for inline panel, cannot add form.");
            return;
        }


        const targetedElement = this.getTargetElement(specifier);
        let targetAppendIndex = this.formsTarget.children.length;
        if (targetedElement) {
            for (let i = 0; i < this.formsTarget.children.length; i++) {
                if (this.formsTarget.children[i] === targetedElement) {
                    targetAppendIndex = i + 1;
                    break;
                }
            }
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

        if (targetAppendIndex >= this.formsTarget.children.length) {
            for (let i = 0; i < newFormElem.children.length; i++) {
                animator.addElement(newFormElem.children[i] as HTMLElement);
                this.formsTarget.appendChild(newFormElem.children[i]);
            }
        } else {
            for (let i = 0; i < newFormElem.children.length; i++) {
                animator.addElement(newFormElem.children[i] as HTMLElement);
                this.formsTarget.insertBefore(
                    newFormElem.children[i],
                    this.formsTarget.children[targetAppendIndex + i],
                );
            }
        }

        animator.start();

        this.lastFormIndex++;
    }

    removeForm(specifier: string | number) {
        const targetedElement = this.getTargetElement(specifier);
        if (!targetedElement) {
            console.error("No targeted element found for inline panel, cannot remove form.");
            return;
        }

        this.formsTarget.removeChild(targetedElement);
    }

    private getTargetElement(specifier: string | number | null): HTMLElement | null {
        let targetedElement: HTMLElement | null = null;
        let typeOfSpecifier = typeof specifier;
        if (specifier === null || typeOfSpecifier === "undefined" || (typeOfSpecifier === "string" && specifier === "")) {
            targetedElement = this.formsTarget.lastElementChild as HTMLElement;
        } else if (typeOfSpecifier === "string") {
            targetedElement = document.getElementById(specifier as string);
        } else if (typeOfSpecifier === "number") {
            if (this.formsTarget.children.length > (specifier as number)) {
                targetedElement = this.formsTarget.children[specifier as number] as HTMLElement;
            } else {
                targetedElement = this.formsTarget.lastElementChild as HTMLElement;
                console.warn(`Index ${specifier} out of bounds, appending to end instead.`);
            }
        }
        return targetedElement;
    }
}

export { InlinePanelController };