<script lang="ts">
	import type { Snippet } from 'svelte';
	// Shared top band for secondary marketing pages: brand colour, curved edge.
	let { title, lead = '', image = '', children }: { title: string; lead?: string; image?: string; children?: Snippet } = $props();
</script>

<section class="relative overflow-hidden bg-primary text-white">
	<div class="mx-auto grid max-w-6xl items-center gap-8 px-4 pt-12 pb-20 sm:px-6 md:grid-cols-[1.2fr_1fr] md:pb-24">
		<div>
			<h1 class="anim-rise text-4xl sm:text-5xl">{title}</h1>
			{#if lead}<p class="anim-rise mt-4 max-w-xl text-lg text-white/85" style="--d:90ms">{lead}</p>{/if}
			{@render children?.()}
		</div>
		{#if image}<img src={image} alt="" class="anim-rise-scale mx-auto hidden max-h-64 w-auto rounded-3xl bg-white p-5 md:block" style="--d:150ms" />{/if}
	</div>
	<svg class="absolute inset-x-0 bottom-0 block h-10 w-full md:h-16" viewBox="0 0 1440 80" preserveAspectRatio="none" aria-hidden="true">
		<path d="M0 80V36C240 6 480-6 720 6s480 50 720 30v44H0Z" fill="var(--color-canvas)" />
	</svg>
</section>
