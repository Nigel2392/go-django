const SVG_NAMESPACE = "http://www.w3.org/2000/svg";

const SVG_TAGS = new Set([
  "svg", "path", "circle", "rect", "g", "defs", "clipPath", "use", "line", "polygon", "polyline",
  "ellipse", "text", "tspan", "marker", "pattern", "mask", "symbol", "view"
]);

export function jsx(
  tag: JSX.Tag | JSX.Component,
  attributes: { [key: string]: any } | null,
  ...children: any[]
) {
  if (typeof tag === "function") {
    return tag(attributes ?? {}, children);
  }

  const isSvg = SVG_TAGS.has(tag);
  const element = isSvg
    ? document.createElementNS(SVG_NAMESPACE, tag)
    : document.createElement(tag);

  if (attributes) {
    for (const [key, value] of Object.entries(attributes)) {
      if (key === "class") {
        element.setAttribute("class", value);
      } else if (key === "style" && typeof value === "object") {
        Object.assign((element as HTMLElement).style, value);
      } else if (isSvg) {
        // Use normal setAttribute for standard SVG attributes
        element.setAttribute(key, value);
      } else if (key in element) {
        (element as any)[key] = value;
      } else {
        element.setAttribute(key, value);
      }
    }
  }

  const appendChildSafely = (child: any) => {
    if (typeof child === "string") {
      element.appendChild(document.createTextNode(child));
    } else if (child instanceof Node) {
      element.appendChild(child);
    } else if (Array.isArray(child)) {
      child.forEach(appendChildSafely);
    } else {
      console.warn("Invalid child type:", child);
    }
  };

  children.forEach(appendChildSafely);

  return element;
}
