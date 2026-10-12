<script lang="ts">
	import type { Snippet } from 'svelte';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	let { title, lead, image, points = [], children }: { title: string; lead: string; image: string; points?: string[]; children: Snippet } = $props();
</script>

<div class="mx-auto grid max-w-6xl gap-6 sm:gap-8 px-3.5 sm:px-6 py-6 sm:py-10 lg:grid-cols-2 lg:py-16">
	<aside class="anim-rise-scale relative hidden overflow-hidden rounded-[2rem] bg-primary p-10 text-white lg:block">
		<p class="text-3xl font-extrabold">Belajar yang terasa seperti bermain</p>
		<ul class="mt-6 space-y-3">
			{#each points as p}<li class="flex gap-3 text-white/90"><CheckCircle size={22} weight="fill" class="shrink-0 text-sun" aria-hidden="true" />{p}</li>{/each}
		</ul>
		<img src={image} alt="" class="anim-float mt-10 w-full max-w-sm rounded-3xl bg-white p-6" />
	</aside>
	<div class="card anim-rise self-start p-4.5 sm:p-8 lg:p-10" style="--d:100ms">
		<h1 class="text-2xl sm:text-3xl font-extrabold text-ink">{title}</h1>
		<p class="mt-1.5 sm:mt-2 text-xs sm:text-sm text-ink-soft">{lead}</p>
		{@render children()}
	</div>
</div>
