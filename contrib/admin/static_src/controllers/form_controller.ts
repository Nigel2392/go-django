import { Controller } from "@hotwired/stimulus";

/**
 * <form data-controller="form"
 *       data-form-confirm-message-value="Unsaved changes will be lost. Continue?"
 *       data-form-watch-events-value='["input","change"]'
 *       data-form-watch-content-editable-value="true"
 *       data-form-link-selector-value='a[href]:not([target]):not([download])'>
 * </form>
 */
class FormController extends Controller<HTMLFormElement> {
    static values = {
        confirmMessage: { type: String },
        watchEvents: { type: Array, default: ["input", "change"] },
        watchContentEditable: { type: Boolean, default: true },
        linkSelector: { type: String, default: 'a[href]:not([target]):not([download])' },
        dirty: { type: Boolean, default: false }
    };

    declare hasConfirmMessageValue: boolean;
    declare confirmMessageValue: string;
    declare watchEventsValue: string[];
    declare watchContentEditableValue: boolean;
    declare linkSelectorValue: string;
    declare dirtyValue: boolean;

    private listeners: Array<() => void> = [];
    private initialSnapshot = "";

    markClean() {
        this.dirtyValue = false;
        // refresh baseline snapshot (use setTimeout to allow any programmatic changes to settle)
        setTimeout(() => (this.initialSnapshot = this.serializeForm(this.element)));
    }

    markDirty() {
        this.dirtyValue = true;
    }

    isDirty() {
        return this.dirtyValue;
    }

    connect() {
        if (!(this.element instanceof HTMLFormElement)) {
            throw new Error("FormController must be attached to a <form> element");
        }

        if (!this.hasConfirmMessageValue) {
            this.confirmMessageValue = window.i18n.gettext("You have unsaved changes.\nAre you sure you want to leave this page?");
        }

        this.initialSnapshot = this.serializeForm(this.element);

        this.setupInputWatchers();
        this.setupContentEditableWatchers();
        this.setupBeforeUnload();
        this.setupLinkInterception();
        this.setupFormLifecycle();
    }

    disconnect() {
        this.teardown();
    }

    private setupInputWatchers() {
        for (const type of this.watchEventsValue) {
            const handler = (e: Event) => {
                if ((e as any).isTrusted === false) return; // ignore synthetic programmatic changes

                const newSnapshot = this.serializeForm(this.element);
                console.log({ newSnapshot, initial: this.initialSnapshot });
                if (newSnapshot === this.initialSnapshot) {
                    this.dirtyValue = false;
                } else {
                    this.markDirty();
                }
            };
            this.element.addEventListener(type, handler as EventListener, { passive: true });
            this.listeners.push(() => this.element.removeEventListener(type, handler as EventListener));
        }
    }

    private setupContentEditableWatchers() {
        if (!this.watchContentEditableValue) return;

        const editables = this.element.querySelectorAll<HTMLElement>('[contenteditable=""], [contenteditable="true"]');
        const handler = () => this.markDirty();
        editables.forEach(el => {
            el.addEventListener("input", handler, { passive: true });
            this.listeners.push(() => el.removeEventListener("input", handler));
        });
    }

    private setupBeforeUnload() {
        const beforeUnload = (e: BeforeUnloadEvent) => {
            if (!this.dirtyValue) return;
            e.preventDefault();
            e.returnValue = true;
            // alert(this.confirmMessageValue);
            return this.confirmMessageValue; // some browsers use this as the prompt text
        };
        window.addEventListener("beforeunload", beforeUnload);
        this.listeners.push(() => window.removeEventListener("beforeunload", beforeUnload));
    }

    private setupLinkInterception() {
        const clickHandler = (e: MouseEvent) => {
            if (!this.dirtyValue) return;

            // Only plain left-clicks without modifiers
            if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;

            // Find nearest anchor
            let a = e.target as HTMLElement | null;
            while (a && a.tagName !== "A") a = a.parentElement;
            const anchor = a as HTMLAnchorElement | null;

            if (!anchor || !anchor.matches(this.linkSelectorValue)) return;

            const ok = window.confirm(this.confirmMessageValue);
            if (!ok) {
                e.preventDefault();
                e.stopPropagation();
            } else {
                // user confirmed navigation: remove guards to avoid double prompts
                this.teardown();
            }
        };

        document.addEventListener("click", clickHandler, true);
        this.listeners.push(() => document.removeEventListener("click", clickHandler, true));
    }

    private setupFormLifecycle() {
        const onSubmit = () => {
            this.dirtyValue = false;
            this.teardown();
        };
        const onReset = () => {
            // after reset, recalc snapshot once the form values have reset
            this.dirtyValue = false;
            setTimeout(() => (this.initialSnapshot = this.serializeForm(this.element)));
        };

        this.element.addEventListener("submit", onSubmit);
        this.element.addEventListener("reset", onReset);

        this.listeners.push(() => this.element.removeEventListener("submit", onSubmit));
        this.listeners.push(() => this.element.removeEventListener("reset", onReset));
    }

    private teardown() {
        while (this.listeners.length) this.listeners.pop()!();
    }

    // --- Serialization (stable & TS-safe) ------------------------------------

    private serializeForm(formEl: HTMLFormElement) {
        const fd = new FormData(formEl);

        // Ensure unchecked checkboxes/radios get represented
        for (const el of formEl.querySelectorAll<HTMLInputElement>('input[type="checkbox"], input[type="radio"]')) {
            if (el.name && !fd.has(el.name)) fd.append(el.name, "");
        }

        // Build stable key/value pairs; normalize File values to a signature string
        const pairs: Array<[string, string]> = [];
        for (const [k, v] of fd.entries()) {
            if (typeof v === "string") {
                pairs.push([k, v]);
            } else {
                // Represent a file by name/size/lastModified (good enough for change detection)
                pairs.push([k, `${v.name}:${v.size}:${v.lastModified}`]);
            }
        }

        // Sort deterministically
        pairs.sort(([aK, aV], [bK, bV]) => (aK === bK ? aV.localeCompare(bV) : aK.localeCompare(bK)));

        return JSON.stringify(pairs);
    }
}

export { FormController };