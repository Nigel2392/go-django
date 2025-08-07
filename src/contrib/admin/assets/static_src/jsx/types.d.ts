declare namespace JSX {
    type Element = HTMLElement;
    type Tag = keyof HTMLElementTagNameMap;

    interface IntrinsicElements extends IntrinsicElementMap, SVGElementMap { }

    type IntrinsicElementMap = {
        [K in keyof HTMLElementTagNameMap]: {
            [k: string]: any;
        }
    }

    type SVGElementMap = {
        [K in keyof SVGElementTagNameMap]: {
            [k: string]: any;
        }
    }
    
    interface Component {
        (properties?: { [key: string]: any }, children?: any[]): Node;
    }

    interface ElementChildrenAttribute {
        children: {};
    }

    interface ElementClass {
        render: any;
    }
}