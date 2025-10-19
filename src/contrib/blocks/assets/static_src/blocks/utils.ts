export function moveUp(el: HTMLElement) {
    const prev = el.previousElementSibling;
    if (prev) {
        el.parentNode.insertBefore(el, prev); // swap positions
    }
}

export function moveDown(el: HTMLElement) {
    const next = el.nextElementSibling;
    if (next) {
        el.parentNode.insertBefore(next, el); // swap by moving "next" before current
    }
}


const nonCopyDisplayAttrs: { [key: string]: boolean } = {
    "data-controller": true,
    "class": true,
    "style": true,
}

export function copyAttrs(from: HTMLElement, has: (name: string) => boolean, setter: (key: string, value: string) => void): void {
    const attrs: { [key: string]: string } = {};
    // placeholder attrs copied
    for (let i = 0; i < from.attributes.length; i++) {
        if (!has(from.attributes[i].name) && !nonCopyDisplayAttrs[from.attributes[i].name]) {
            attrs[from.attributes[i].name] = from.attributes[i].value;
        }
    }

    Object.keys(attrs).forEach((key) => {
        setter(key, attrs[key]);
    });
}
