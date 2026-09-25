<script lang="ts">
	import { get } from '$lib/api';
	import Avatar from '$lib/components/Avatar.svelte';
	import Wordmark from '$lib/components/Wordmark.svelte';
	import friends from '$lib/illustrations/friends.svg';
	import type { Child } from '$lib/types';
	let children = $state<Child[] | null>(null);
	get('/children').then((r) => (children = r.children));
</script>

<svelte:head><title>Siapa yang main? | EduPlay</title></svelte:head>

<main class="flex min-h-dvh flex-col bg-primary px-4 py-8 text-white">
	<div class="mx-auto flex w-full max-w-2xl flex-1 flex-col">
		<Wordmark light href="/app/play" />
		<img src={friends} alt="" class="anim-pop mx-auto mt-8 w-44 rounded-3xl bg-white p-4" />
		<h1 class="anim-rise mt-6 text-center text-4xl" style="--d:120ms">Siapa yang mau main?</h1>
		{#if children}
			{#if children.length === 0}
				<p class="mt-8 text-center">Belum ada profil anak. <a class="font-bold text-sun underline" href="/app/parent/children/new">Buat profil</a></p>
			{:else}
				<ul class="mt-8 grid grid-cols-2 gap-4 sm:grid-cols-3">
					{#each children as c, i (c.id)}
						<li class="anim-rise" style="--d:{200 + i * 90}ms">
							<a href="/app/play/{c.id}" class="flex flex-col items-center gap-3 rounded-3xl bg-white px-3 py-6 text-ink shadow-[0_6px_0_rgb(0_0_0/0.15)] transition-transform active:translate-y-1">
								<Avatar code={c.avatar} size={84} />
								<span class="text-xl font-extrabold">{c.nickname}</span>
								<span class="pill bg-primary-tint text-primary-deep">Level {c.level}</span>
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		{/if}
		<a href="/app/parent" class="mt-auto self-center pt-10 font-semibold text-white/80 underline">Area orang tua</a>
	</div>
</main>
