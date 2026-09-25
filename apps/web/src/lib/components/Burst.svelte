<script lang="ts">
	// A small celebratory burst of stars and dots around its parent (which must be
	// position: relative). 12 elements, CSS-only, ~0.8s, then gone.
	let { count = 12, spread = 70 }: { count?: number; spread?: number } = $props();
	import { untrack } from 'svelte';
	const colors = ['#ffc857', '#5b5ce2', '#54c98a', '#ff8a4c', '#e5579a', '#3a8ee8'];
	// Particles are generated once per burst.
	const bits = untrack(() => Array.from({ length: count }, (_, i) => {
		const angle = (i / count) * Math.PI * 2 + Math.random() * 0.4;
		const dist = spread * (0.7 + Math.random() * 0.5);
		return {
			x: Math.cos(angle) * dist,
			y: Math.sin(angle) * dist,
			c: colors[i % colors.length],
			star: i % 3 === 0,
			s: 6 + Math.random() * 6,
			d: Math.random() * 60
		};
	}));
</script>

<span class="pointer-events-none absolute inset-0 grid place-items-center" aria-hidden="true">
	{#each bits as b}
		<span class="bit" style="--x:{b.x}px;--y:{b.y}px;--d:{b.d}ms;width:{b.s * (b.star ? 1.6 : 1)}px;height:{b.s * (b.star ? 1.6 : 1)}px;background:{b.c}" class:star={b.star}></span>
	{/each}
</span>

<style>
	.bit {
		position: absolute;
		border-radius: 999px;
		opacity: 0;
		animation: fly 0.8s cubic-bezier(0.22, 1, 0.36, 1) var(--d) both;
	}
	.star {
		border-radius: 0;
		clip-path: polygon(50% 0, 61% 35%, 98% 35%, 68% 57%, 79% 91%, 50% 70%, 21% 91%, 32% 57%, 2% 35%, 39% 35%);
	}
	@keyframes fly {
		0% { opacity: 1; transform: translate(0, 0) scale(0.4) rotate(0); }
		70% { opacity: 1; }
		100% { opacity: 0; transform: translate(var(--x), var(--y)) scale(1) rotate(160deg); }
	}
	@media (prefers-reduced-motion: reduce) {
		.bit { display: none; }
	}
</style>
