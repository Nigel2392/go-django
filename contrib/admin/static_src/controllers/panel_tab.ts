import { Controller, ActionEvent } from "@hotwired/stimulus";

class TabPanelController extends Controller<HTMLElement> {
    static targets = [
      "tab", "control",
    ];

    declare listeners: (() => void)[];
    declare hasParentTab: boolean;
    declare tabTargets: HTMLElement[];
    declare controlTargets: HTMLElement[];

    connect() {
        this.listeners = [];
        this.tabTargets[0].classList.add("active");
        this.controlTargets[0].classList.add("active");
        this.hasParentTab = this.element.parentElement.closest(`[data-controller="${this.identifier}"]`) !== null;

        if (!this.hasParentTab && window.location.hash && window.location.hash.startsWith("#tab-")) {
            const index = parseInt(window.location.hash.split("-").pop() || "0", 10);
            this.selectTab(index, false);
        }

        this.setupValidation();
    }

    disconnect(): void {
        this.listeners.forEach((listener) => { listener() })
    }

    select(event: ActionEvent) {
        event.preventDefault();

        if (event.params.index === undefined || event.params.index >= this.tabTargets.length) {
            console.warn("Invalid tab index:", event.params.index, this.tabTargets);
            return;
        }

        this.selectTab(parseInt(event.params.index, 10));
    }

    selectTab(index: number, setHash: boolean = true) {

        if (!this.hasParentTab && setHash) {
            window.location.hash = `#tab-${index}`;
        }

        this.tabTargets.forEach((tab, i) => {
            tab.classList.toggle("active", i === index);
        });
        this.controlTargets.forEach((control, i) => {
            control.classList.toggle("active", i === index);
        });
    }

    markTab(index: number, marked: boolean = true) {

        if (index >= this.tabTargets.length || index >= this.controlTargets.length) {
            throw new Error(`index ${index} out of range for ${min(this.tabTargets.length, this.controlTargets.length)}`)
        }

        this.tabTargets[index].classList.toggle("tab-error", marked);
        this.controlTargets[index].classList.toggle("tab-error", marked);
    }

    private setupValidation() {
        // mark tabs invalid if it contains invalid inputs
        const onInvalid = (e: Event) => {
            // check if input is present in root tab element
            if (!this.element.contains(e.target as Node)) return;

            e.preventDefault();

            // find invalid field's tab
            const targetTab = this.tabTargets.find(tab => tab.contains(e.target as Node));
            if (targetTab) {
                const index = this.tabTargets.indexOf(targetTab);
                this.markTab(index, true);
            }
        };

        this.element.addEventListener("invalid", onInvalid, true);
        this.listeners.push(() => this.element.removeEventListener("invalid", onInvalid, true));

        // clear all errors when form is resubmitted
        const form = this.element.closest("form");
        if (form) {
            const clearErrors = (e: Event) => {
                const target = e.target as HTMLElement;
                const isSubmitClick = e.type === "click" && target.closest('button:not([type="button"]), input[type="submit"]');
                const isEnterKey = e.type === "keydown" && (e as KeyboardEvent).key === "Enter" && target.tagName !== "TEXTAREA";

                if (isSubmitClick || isEnterKey) {
                    this.tabTargets.forEach((_, index) => this.markTab(index, false));
                }
            };

            // Bind to form in capture phase to clear errors right before native validation runs
            form.addEventListener("click", clearErrors, true);
            form.addEventListener("keydown", clearErrors, true);
            this.listeners.push(() => form.removeEventListener("click", clearErrors, true));
            this.listeners.push(() => form.removeEventListener("keydown", clearErrors, true));
        }
    }
}

function min(...all: number[]) {
    var c = null;
    for (let i = 0; i < all.length; i++) {
        if (c == null) {
            c = all[i];
            continue
        }

        if (all[i] < c) {
            c = all[i]
        }
    }
    return c
}

export { TabPanelController };