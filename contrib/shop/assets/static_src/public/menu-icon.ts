const template = document.createElement("template");

template.innerHTML = `
    <style>
        :host {
            display: inline-flex;
            vertical-align: middle;
            color: inherit;
        }

        button {
            display: inline-grid;
            place-items: center;
            width: var(--menu-icon-size, 44px);
            height: var(--menu-icon-size, 44px);
            padding: 0;
            border: 0;
            border-radius: var(--menu-icon-radius, 8px);
            background: var(--menu-icon-background, transparent);
            color: inherit;
            cursor: pointer;
            -webkit-tap-highlight-color: transparent;
        }

        button:hover:not(:disabled) {
            background: var(--menu-icon-hover-background, rgb(127 127 127 / 14%));
        }

        button:focus-visible {
            outline: 2px solid currentColor;
            outline-offset: 3px;
        }

        button:disabled {
            opacity: 0.45;
            cursor: default;
        }

        .icon {
            position: relative;
            display: block;
            width: 22px;
            height: 18px;
        }

        .bar {
            position: absolute;
            left: 0;
            width: 100%;
            height: 2px;
            border-radius: 1px;
            background: currentColor;
            transform-origin: center;
            transition: transform 160ms ease, opacity 160ms ease;
        }

        .bar:first-child { top: 0; }
        .bar:nth-child(2) { top: 8px; }
        .bar:last-child { top: 16px; }

        :host([expanded]) .bar:first-child {
            transform: translateY(8px) rotate(45deg);
        }

        :host([expanded]) .bar:nth-child(2) {
            opacity: 0;
        }

        :host([expanded]) .bar:last-child {
            transform: translateY(-8px) rotate(-45deg);
        }

        @media (prefers-reduced-motion: reduce) {
            .bar { transition: none; }
        }
    </style>

    <button type="button" aria-expanded="false" aria-label="Open menu">
        <span class="icon" aria-hidden="true">
            <span class="bar"></span>
            <span class="bar"></span>
            <span class="bar"></span>
        </span>
    </button>
`;

export interface MenuToggleDetail {
    expanded: boolean;
}

export class MenuIcon extends HTMLElement {
    static get observedAttributes(): string[] {
        return ["expanded", "label", "disabled"];
    }

    private readonly button: HTMLButtonElement;

    constructor() {
        super();

        const root = this.attachShadow({ mode: "open" });
        root.appendChild(template.content.cloneNode(true));

        this.button = root.querySelector("button")!;
        this.button.addEventListener("click", () => {
            this.expanded = !this.expanded;

            this.dispatchEvent(new CustomEvent<MenuToggleDetail>("menu-toggle", {
                bubbles: true,
                composed: true,
                detail: { expanded: this.expanded },
            }));
        });
    }

    get expanded(): boolean {
        return this.hasAttribute("expanded");
    }

    set expanded(value: boolean) {
        this.toggleAttribute("expanded", value);
    }

    connectedCallback(): void {
        this.updateButton();
    }

    attributeChangedCallback(): void {
        this.updateButton();
    }

    protected get defaultLabel(): string {
        return this.expanded ? "Close menu" : "Open menu";
    }

    protected updateButton(): void {
        const label = this.getAttribute("label")
            || this.defaultLabel;

        this.button.setAttribute("aria-expanded", String(this.expanded));
        this.button.setAttribute("aria-label", label);
        this.button.disabled = this.hasAttribute("disabled");
    }
}

if (!customElements.get("django-menu-icon")) {
    customElements.define("django-menu-icon", MenuIcon);
}
