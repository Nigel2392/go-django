import { PanelController } from "../controllers/panel";
import { jsx } from "../jsx";
import Icon from "./icon";

export function PanelHeading(panelId?: string, allowPanelLink: boolean = true, ...children: any[]) {
    return (
      <div class="panel__heading" data-panel-target="heading">
          {panelId && allowPanelLink ? (
            <a class="panel__icon" data-panel-target="linkIcon" href={`#${panelId}`}>
              { Icon('icon-link', { class: 'link-logo' }) }
            </a>
          ) : null}

          <div class="panel__title" data-action="click->panel#toggle" {...(panelId ? { id: panelId } : {})}>
              { children }
          </div>
      </div>
    );
}

export function PanelErrors(errors?: any[] | HTMLElement | null, ...children: any[]) {
    if (!errors || Array.isArray(errors) && errors.length === 0) return null;

    const errorsContainer = ( <div class="panel__errors"></div>)
    if (errors instanceof HTMLElement) {
        errorsContainer.appendChild(errors);
        return errorsContainer;
    }

    if (Array.isArray(errors)) {
        errors.forEach((error) => {
            const errorItem = (
                <li class="panel__error">{error instanceof Error ? error.message : String(error)}</li>
            );
            errorsContainer.appendChild(errorItem);
        });
    } else {
        const errorItem = (
            <li class="panel__error">{(errors as any) instanceof Error ? (errors as Error).message : String(errors)}</li>
        );
        errorsContainer.appendChild(errorItem);
    }

    return errorsContainer;
}

export function PanelHelpText(helptext: any) {
    if (helptext === null || helptext === undefined) return null;
    return (
        <div class="panel__help">
            { typeof helptext === "function" ? helptext() : helptext }
        </div>
    );
}

export function PanelBody(...children: any[]) {
    return (
        <div class="panel__body" data-panel-target="content">
            { children }
        </div>
    );
}

function isZero(value: any): boolean {
    if (value === null || value === undefined) return true;
    if (typeof value === "string" && value.trim() === "") return true;
    if (Array.isArray(value) && value.length === 0) return true;
    return false;
}

export function getPanelAttrs(attrs?: Record<string, any>, opts?: { panelId?: string; inputId?: string; hidden?: boolean, class?: string }): Record<string, any> {
    const root: any = { ...(attrs || {}) };
    const classes: string[] = ["panel", "collapsible"];
    if (root.class) classes.push(root.class);
    if (opts?.class) classes.push(opts.class);

    let controllerAttribute = root["data-controller"] || "";
    if (!controllerAttribute) {
        controllerAttribute = "panel";
    } else if (!controllerAttribute.split(" ").includes("panel")) {
        controllerAttribute += " panel";
    }
    root["data-controller"] = controllerAttribute;
    
    if (opts?.panelId) root["data-panel-panel-value"] = opts.panelId;
    if (opts?.inputId) root["data-panel-input-id"] = opts.inputId;

    if (opts?.hidden) {
        classes.push("collapsed");
    }

    root.class = classes.join(" ");
    return root;
}

export type PanelOptions = {
    class?: string;
    panelId?: string;
    hidden?: boolean;
    allowPanelLink?: boolean;
    inputId?: string;
    attrs?: Record<string, any>;
    heading?: any;
    helpText?: any;
    errors?: any[] | HTMLElement;
    children?: any;
}

export type Panel = HTMLElement & {
    opts: PanelOptions;
    heading: HTMLElement | null;
    helpText: HTMLElement | null;
    errors: HTMLElement | null;
    body: HTMLElement;
    toggle(): void;
    show(): void;
    hide(): void;
}

export function PanelComponent(props: PanelOptions) {
    const {
        class: cls,
        panelId,
        hidden,
        allowPanelLink,
        inputId,
        attrs,
        heading,
        helpText,
        errors,
        children,
    } = props;

    // Create panel elements
    const panel = ( <div {...getPanelAttrs(attrs, { panelId, inputId, hidden, class: cls })}></div> ) as Panel;
    const panelHeading = (!hidden && !isZero(heading)) ? PanelHeading(panelId, allowPanelLink !== false, heading) : null;
    const panelErrors = PanelErrors(errors);
    const panelHelpText = (!hidden && !isZero(helpText)) ? PanelHelpText(helpText) : null;
    const panelBody = Array.isArray(children) ? PanelBody(...children) : PanelBody(children);
    
    // Set panel options
    panel.opts = props;

    // Append elements
    if (panelHeading) panel.heading = panel.appendChild(panelHeading);
    if (panelHelpText) panel.helpText = panel.appendChild(panelHelpText);
    if (panelErrors) panel.errors = panel.appendChild(panelErrors);
    panel.body = panel.appendChild(panelBody);

    // Setup panel shortcut functions
    panel.toggle = function() {
        const controller = window.Stimulus.getControllerForElementAndIdentifier(this, "panel") as PanelController;
        if (controller) controller.toggle();
    }
    panel.show = function() {
        const controller = window.Stimulus.getControllerForElementAndIdentifier(this, "panel") as PanelController;
        if (controller) controller.collapse(false);
    }
    panel.hide = function() {
        const controller = window.Stimulus.getControllerForElementAndIdentifier(this, "panel") as PanelController;
        if (controller) controller.collapse(true);
    }

    return panel;
}

