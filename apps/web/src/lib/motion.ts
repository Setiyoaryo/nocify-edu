// Tiny motion helpers. No animation library: everything runs on CSS
// transforms/opacity, and the global prefers-reduced-motion rule turns it off.

/** Adds `is-in` once the element scrolls into view (for .reveal / .reveal-stagger). */
export function reveal(node: HTMLElement, opts: { threshold?: number } = {}) {
	if (typeof IntersectionObserver === 'undefined') {
		node.classList.add('is-in');
		return {};
	}
	const io = new IntersectionObserver(
		(entries) => {
			for (const e of entries) {
				if (e.isIntersecting) {
					node.classList.add('is-in');
					io.disconnect();
				}
			}
		},
		{ threshold: opts.threshold ?? 0.15, rootMargin: '0px 0px -40px 0px' }
	);
	io.observe(node);
	return { destroy: () => io.disconnect() };
}

export function prefersReducedMotion() {
	return typeof matchMedia !== 'undefined' && matchMedia('(prefers-reduced-motion: reduce)').matches;
}
