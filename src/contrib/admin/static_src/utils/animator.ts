
export type openAnimatorOptions = {
    elements?: HTMLElement | HTMLElement[];
    duration?: number;
    easing?: string;
    animFrom?: { [key: string]: string };
    animTo?: { [key: string]: string };
    onAdded?: (elem: HTMLElement) => void;
    onStart?: (elem: HTMLElement) => void;
    onFinished?: (elem: HTMLElement) => void;
}

export class openAnimator {
    elements: HTMLElement[];
    options: openAnimatorOptions;

    constructor(options: openAnimatorOptions = { duration: 300, easing: "ease" }) {
        this.options = options;
    
        if (options.elements) {
            if (options.elements instanceof HTMLElement) {
                this.elements = [options.elements];
            } else {
                this.elements = options.elements;
            }
        } else {
            this.elements = [];
        }
    }

    addElement(...elements: HTMLElement[]) {
        elements.forEach(elem => {
            this.options.onAdded?.(elem);
        });
        this.elements.push(...elements);
    }

    start() {
        this.elements.forEach(elem => {
            this.options.onStart?.(elem);

            // Start the animation on the next frame so start/end are distinct
            requestAnimationFrame(() => {
                const anim = elem.animate(
                    [
                        { height: '0px', ...this.options.animFrom },
                        { height: `${elem.offsetHeight}px`, ...this.options.animTo },
                    ],
                    {
                        duration: this.options.duration || 300,
                        easing: this.options.easing || 'ease',
                        fill: 'none', // don't retain the final numeric height
                    }
                );
              
                anim.onfinish = () => {
                    this.options.onFinished?.(elem);
                };
            });
        });
    }
}
