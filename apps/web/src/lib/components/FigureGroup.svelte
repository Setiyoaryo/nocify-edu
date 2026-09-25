<script lang="ts">
	import Figure from './Figure.svelte';
	import { figure, colorName } from '$lib/figures';
	export type Fig = { icon: string; count?: number; color?: string };
	// Renders a row of pictured objects, e.g. 3 oranges then a question mark,
	// with a single text alternative for screen readers.
	let { figures = [], size = 44, class: klass = '' }: { figures: Fig[]; size?: number; class?: string } = $props();
	const alt = $derived(
		figures.map((f) => `${f.count && f.count > 1 ? f.count + ' ' : ''}${figure(f.icon).label}${f.color ? ' ' + colorName[f.color] : ''}`).join(', ')
	);
</script>

<span class="flex flex-wrap items-center justify-center gap-1.5 {klass}" role="img" aria-label={alt}>
	{#each figures as f}
		{#each Array(Math.max(1, f.count ?? 1)) as _}
			<Figure name={f.icon} color={f.color} {size} />
		{/each}
	{/each}
</span>
