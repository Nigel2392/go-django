export function jsx(
    tag: JSX.Tag | JSX.Component,
    attributes: { [key: string]: any } | null,
    ...children: any[]
) {
    if (typeof tag === 'function') {
        return tag(attributes ?? {}, children);
    }

    const element = document.createElement(tag);

    // Assign attributes
    if (attributes) {
        for (const [key, value] of Object.entries(attributes)) {
            if (key in element) {
                (element as any)[key] = value;
            } else {
                element.setAttribute(key, value);
            }
        }
    }

    const appendChildSafely = (child: any) => {
        if (typeof child === 'string') {
            element.appendChild(document.createTextNode(child));
        } else if (child instanceof Node) {
            element.appendChild(child);
        } else if (Array.isArray(child)) {
            child.forEach(appendChildSafely);
        } else {
            console.warn('Invalid child type:', child);
        }
    };

    children.forEach(appendChildSafely);

    return element;
}