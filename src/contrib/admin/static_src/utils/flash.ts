type FlashOptions = {
    color?: string;
    duration?: number;
    iters?: number;
    delay?: number;
}

export default function(element: Element | null, opts: FlashOptions = { color: 'red', duration: 200, iters: 2, delay: 0 }) {
    if (!element) return;
    element.animate(
        [
            { boxShadow: `0 0 0px ${opts.color}` },
            { boxShadow: `0 0 10px ${opts.color}` },
            { boxShadow: `0 0 0px ${opts.color}` },
        ],
        {
            duration: opts.duration,
            easing: "linear",
            fill: 'none',
            iterations: opts.iters,
            delay: opts.delay,
            direction: 'alternate',
        }
    );
}

export {
    FlashOptions,
}
