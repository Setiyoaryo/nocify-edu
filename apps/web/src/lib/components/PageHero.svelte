<script lang="ts">
	import type { Snippet } from 'svelte';
	// Shared top band for secondary marketing pages: brand colour, curved edge,
	// illustration on a soft disc (same language as the landing hero).
	let { title, lead = '', image = '', children }: { title: string; lead?: string; image?: string; children?: Snippet } = $props();
</script>

<section class="relative overflow-hidden bg-primary text-white">
	<div class="mx-auto grid max-w-6xl items-center gap-8 px-4 pt-10 pb-20 sm:px-6 md:grid-cols-[1.25fr_1fr] md:pt-14 md:pb-28">
		<div>
			<h1 class="anim-rise text-[2.2rem] leading-tight sm:text-5xl">{title}</h1>
			{#if lead}<p class="anim-rise mt-4 max-w-xl text-base text-white/85 sm:text-lg" style="--d:90ms">{lead}</p>{/if}
			{@render children?.()}
		</div>
		{#if image}
			<div class="anim-rise-scale relative mx-auto hidden size-72 md:block lg:size-80" style="--d:150ms" aria-hidden="true">
				<div class="absolute inset-0 rounded-full bg-white/8"></div>
				<div class="absolute inset-7 rounded-full bg-white/10"></div>
				<div class="absolute inset-14 grid place-items-center rounded-full bg-white shadow-[0_24px_50px_-20px_rgb(10_10_40/0.5)]">
					<img src={image} alt="" class="anim-float w-[78%]" />
				</div>
			</div>
		{/if}
	</div>
	<svg class="absolute inset-x-0 bottom-0 block h-10 w-full md:h-16" viewBox="0 0 1440 80" preserveAspectRatio="none" aria-hidden="true">
		<path d="M0 80V36C240 6 480-6 720 6s480 50 720 30v44H0Z" fill="var(--color-canvas)" />
	</svg>
</section>
