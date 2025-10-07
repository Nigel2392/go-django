
/**
 * Given an element or a NodeList, return the first element that matches the selector.
 * This can be the top-level element itself or a descendant.
 * 
 * Copyright (c) 2014-present Torchbox Ltd and individual contributors.
 * All rights reserved.
 */
export const querySelectorIncludingSelf = (elementOrNodeList: any, selector: any): HTMLElement | null => {
  // if elementOrNodeList not iterable, it must be a single element
  const nodeList = elementOrNodeList.forEach
    ? elementOrNodeList
    : [elementOrNodeList];

  for (let i = 0; i < nodeList.length; i += 1) {
    const container = nodeList[i];
    if (container.nodeType === Node.ELEMENT_NODE) {
      // Check if the container itself matches the selector
      if (container.matches(selector)) {
        return container;
      }

      // If not, search within the container
      const found = container.querySelector(selector);
      if (found) {
        return found;
      }
    }
  }

  return null; // No matching element found
};

class InputNotFoundError extends Error {
    constructor(name: string) {
        super(`No input found with name "${name}"`);
        this.name = 'InputNotFoundError';
    }
}

class BoundWidget<T extends HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement | HTMLButtonElement = HTMLInputElement> {
    input: T | null;
    idForLabel: string | null = null;

    
    constructor(element: HTMLElement | (HTMLElement | ChildNode)[], name: string) {
        const selector = `:is(input,select,textarea,button)[name="${name}"]`;
        this.input = querySelectorIncludingSelf(element, selector) as T | null;
        if (!this.input) {
          throw new InputNotFoundError(name);
        }

        this.idForLabel = this.input.id;
    }

    getState(): any {
        return this.input.value;
    }

    getValue(): any {
        return this.input.value;
    }

    setState(state: any): void {
        this.input!.value = state;
    }

    setValue(value: any): void {
        this.input!.value = value;
    }

    focus(): void {
        this.input!.focus();
    }

    getValueForLabel(): any {
        return this.getValue();
    }

    getTextLabel(opts?: { maxLength?: number }): string | null {
        const val = this.getValueForLabel();
        
        const allowedTypes = ['string', 'number', 'boolean'];
        if (!allowedTypes.includes(typeof val)) {
            return null;
        };

        const valString = String(val).trim();
        const maxLength = opts && opts.maxLength;
        if (maxLength && valString.length > maxLength) {
            return valString.substring(0, maxLength - 1) + '…';
        }
        return valString;
    }
}

type BoundWidgetType = {
    new(element: HTMLElement | (HTMLElement | ChildNode)[], name: string): BoundWidget;
    setState(state: any): void;
    setValue(value: any): void;
    getState(): any;
    getValue(): any;
    getValueForLabel(): any;
    getTextLabel(opts?: { maxLength?: number }): string | null;
    focus(): void;
};

class Widget {
    html: string;
    boundWidgetClass: BoundWidgetType = BoundWidget as BoundWidgetType;

    constructor(html: string) {
        this.html = html;
    }

    render(placeholder: HTMLElement, name: string, id: string, initialState: any, options: any = {}): BoundWidget {
        // Render the widget HTML with name and id replacements
        const html = this.html.
            replace(/__NAME__/g, name).
            replace(/__ID__/g, id);

        
        const temp = document.createElement('div');
        temp.innerHTML = html;

        const children = Array.from(temp.childNodes);
        placeholder.replaceWith(...children);

        const childElements = children.filter(
          (node) => node.nodeType === Node.ELEMENT_NODE,
        );

        if (typeof options?.attributes === 'object') {
            for (const [key, value] of Object.entries(options.attributes)) {
                (childElements[0] as HTMLElement).setAttribute(key, value as string);
            }
        }

        const boundWidget = new this.boundWidgetClass(
            childElements.length === 1 ? childElements[0] as HTMLElement : children,
            name,
        );

        boundWidget.setState(initialState);
        return boundWidget;
    }
}

class BoundCheckboxInput extends BoundWidget {
    getState(): any {
        return this.input.checked;
    }
    getValue(): any {
        return this.input.checked;
    }
    setState(state: any): void {
        this.input.checked = Boolean(state);
    }
    setValue(value: any): void {
        this.input.checked = Boolean(value);
    }
    getValueForLabel(): any {
        return this.getValue() ? window.i18n.gettext('Yes') : window.i18n.gettext('No');
    }
}

class CheckboxInput extends Widget {
    boundWidgetClass = BoundCheckboxInput as BoundWidgetType;
}

class BoundRadioSelect {
    element: HTMLElement;
    name: string;
    idForLabel: string;
    isMultiple: boolean;
    selector: string;

    constructor(element: HTMLElement, name: string) {
        this.element = element;
        this.name = name;
        this.idForLabel = '';
        this.isMultiple = !!this.element.querySelector(
          `input[name="${name}"][type="checkbox"]`,
        );
        this.selector = `input[name="${name}"]:checked`;
    }

    getValueForLabel() {
        const getLabels = (input: HTMLInputElement) => {
            const labels = Array.from(input?.labels || [])
              .map((label) => label.textContent.trim())
              .filter(Boolean);
            return labels.join(', ');
        };
        if (this.isMultiple) {
            return Array.from(this.element.querySelectorAll(this.selector))
              .map(getLabels)
              .join(', ');
        }
        return getLabels(this.element.querySelector(this.selector));
    }

    getTextLabel() {
        // This class does not extend BoundWidget, so we don't have the truncating
        // logic without duplicating the code here. Skip it for now.
        return this.getValueForLabel();
    }

    getValue() {
        if (this.isMultiple) {
            return Array.from(this.element.querySelectorAll<HTMLInputElement>(this.selector)).map(
                (el) => el.value,
            );
        }
        return this.element.querySelector<HTMLInputElement>(this.selector)?.value;
    }

    getState() {
        return Array.from(this.element.querySelectorAll<HTMLInputElement>(this.selector)).map(
            (el) => el.value,
        );
    }

    setState(state: any) {
        const inputs = this.element.querySelectorAll<HTMLInputElement>(`input[name="${this.name}"]`);
        for (let i = 0; i < inputs.length; i += 1) {
            inputs[i].checked = state.includes(inputs[i].value);
        }
    }

    setInvalid(invalid: boolean) {
        this.element
        .querySelectorAll(`input[name="${this.name}"]`)
        .forEach((input) => {
            if (invalid) {
                input.setAttribute('aria-invalid', 'true');
            } else {
                input.removeAttribute('aria-invalid');
            }
        });
    }

    focus() {
        this.element.querySelector<HTMLInputElement>(`input[name="${this.name}"]`)?.focus();
    }
}

class RadioSelect extends Widget {
    boundWidgetClass = BoundRadioSelect as unknown as BoundWidgetType;
}

class BoundSelect extends BoundWidget<HTMLSelectElement> {
    getValueForLabel() {
        return Array.from(this.input.selectedOptions)
        .map((option) => option.text)
        .join(', ');
    }

    getValue() {
        if (this.input.multiple) {
            return Array.from(this.input.selectedOptions).map(
                (option) => option.value,
            );
        }
        return this.input.value;
    }

    getState() {
        return Array.from(this.input.selectedOptions).map((option) => option.value);
    }

    setState(state: any) {
        const options = this.input.options;
        for (let i = 0; i < options.length; i += 1) {
            options[i].selected = state.includes(options[i].value);
        }
    }
}

class Select extends Widget {
    boundWidgetClass = BoundSelect as unknown as BoundWidgetType;
}

export {
    Widget,
    BoundWidget,
    InputNotFoundError,
    CheckboxInput,
    RadioSelect,
    Select,
    BoundCheckboxInput,
    BoundRadioSelect,
    BoundSelect,
}