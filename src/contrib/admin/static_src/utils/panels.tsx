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

export function PanelErrors(errors?: any[] | null, ...children: any[]) {
    if (!errors || errors.length === 0) return null;
    return (
        <div class="panel__errors">
            <ul>
                {errors.map((err) => (
                    <li class="panel__error">
                        {err instanceof Error ? err.message : String(err)}
                    </li>
                ))}
                {children}
            </ul>
        </div>
    );
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

    root["data-controller"] = "panel";
    
    if (opts?.panelId) root["data-panel-panel-value"] = opts.panelId;
    if (opts?.inputId) root["data-panel-input-id"] = opts.inputId;

    if (opts?.hidden) {
        classes.push("collapsed");
    }

    root.class = classes.join(" ");
    return root;
}

export function PanelComponent(props: {
    class?: string;
    panelId?: string;
    hidden?: boolean;
    allowPanelLink?: boolean;
    inputId?: string;
    attrs?: Record<string, any>;
    heading?: any;
    helpText?: any;
    errors?: any[];
    children?: any;
}) {
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


    return (
        <div {...getPanelAttrs(attrs, { panelId, inputId, hidden, class: cls })}>
            { (!hidden && !isZero(heading)) ? PanelHeading(panelId, allowPanelLink !== false, heading) : null }
            { PanelErrors(errors) }
            { (!hidden && !isZero(helpText)) ? PanelHelpText(helpText) : null }
            { Array.isArray(children) ? PanelBody(...children) : PanelBody(children) }
        </div>
    );
}

