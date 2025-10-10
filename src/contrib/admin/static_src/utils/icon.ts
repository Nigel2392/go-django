export default function Icon(name: string, attrs: { [key: string]: string } = {}) {
    const ns = "http://www.w3.org/2000/svg";
    const icon = document.createElementNS(ns, "svg");
    icon.setAttribute("class", `icon ${name}`);

    const use = document.createElementNS(ns, "use");
    use.setAttributeNS("http://www.w3.org/1999/xlink", "href", `#${name}`);
    icon.appendChild(use);

    for (const attr in attrs) {
        if (attr == "class") {
            icon.setAttribute(attr, `icon ${name} ${attrs[attr]}`);
        } else {
            icon.setAttribute(attr, attrs[attr]);
        }
    }
    
    return icon;
}
