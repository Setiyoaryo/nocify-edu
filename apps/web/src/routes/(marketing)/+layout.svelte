<script lang="ts">
	import Wordmark from '$lib/components/Wordmark.svelte';
	import { page } from '$app/state';
	import MobileCta from '$lib/components/MobileCta.svelte';
	import List from 'phosphor-svelte/lib/List';
	import X from 'phosphor-svelte/lib/X';
	let { children } = $props();
	let open = $state(false);
	const links = [
		{ href: '/subjects', label: 'Pelajaran' },
		{ href: '/features', label: 'Cara kerja' },
		{ href: '/for-parents', label: 'Untuk orang tua' },
		{ href: '/pricing', label: 'Harga' },
		{ href: '/faq', label: 'FAQ' }
	];
	$effect(() => {
		page.url.pathname;
		open = false;
	});
</script>

<header class="sticky top-0 z-30 border-b border-line/70 bg-paper/95 backdrop-blur supports-[backdrop-filter]:bg-paper/85">
	<nav class="mx-auto flex h-16 max-w-6xl items-center justify-between gap-4 px-4 sm:px-6" aria-label="Utama">
		<Wordmark />
		<ul class="hidden items-center gap-1 lg:flex">
			{#each links as l}
				<li>
					<a href={l.href} aria-current={page.url.pathname === l.href ? 'page' : undefined}
						class="rounded-full px-3.5 py-2 text-[0.95rem] font-semibold text-ink-soft transition-colors hover:text-ink aria-[current=page]:bg-primary-tint aria-[current=page]:text-primary-deep">{l.label}</a>
				</li>
			{/each}
		</ul>
		<div class="flex items-center gap-2">
			<a href="/login" class="btn btn-quiet btn-sm hidden min-h-11 sm:inline-flex">Masuk</a>
			<a href="/register" class="btn btn-sun btn-sm min-h-11">Daftar gratis</a>
			<button class="grid size-11 place-items-center rounded-full hover:bg-paper-deep lg:hidden" aria-expanded={open} aria-controls="mnav" onclick={() => (open = !open)}>
				<span class="sr-only">Menu</span>
				{#if open}<X size={24} weight="bold" aria-hidden="true" />{:else}<List size={24} weight="bold" aria-hidden="true" />{/if}
			</button>
		</div>
	</nav>
	{#if open}
		<ul id="mnav" class="border-t border-line bg-paper px-4 pt-2 pb-4 lg:hidden">
			{#each [...links, { href: '/login', label: 'Masuk' }] as l}
				<li><a href={l.href} class="block rounded-xl px-3 py-3 font-semibold hover:bg-paper-deep">{l.label}</a></li>
			{/each}
		</ul>
	{/if}
</header>

<main>{@render children()}</main>
<MobileCta />

<footer class="bg-night text-white">
	<div class="mx-auto grid max-w-6xl gap-10 px-4 py-14 sm:px-6 md:grid-cols-[1.4fr_1fr_1fr]">
		<div>
			<Wordmark light />
			<p class="mt-4 max-w-sm text-white/75">Latihan pelajaran sekolah untuk anak TK dan SD, dengan laporan perkembangan per skill untuk orang tua.</p>
		</div>
		<div>
			<p class="font-bold">Jelajahi</p>
			<ul class="mt-3 space-y-2 text-white/75">
				{#each links as l}<li><a class="inline-block py-1.5 hover:text-white" href={l.href}>{l.label}</a></li>{/each}
			</ul>
		</div>
		<div>
			<p class="font-bold">Data anak aman</p>
			<p class="mt-3 text-white/75">Diproses atas persetujuan orang tua/wali, hanya untuk kebutuhan belajar. Tanpa iklan untuk anak.</p>
		</div>
	</div>
	<div class="border-t border-white/10">
		<p class="mx-auto max-w-6xl px-4 pt-5 pb-28 text-sm text-white/60 sm:px-6 md:pb-5">© 2026 EduPlay. Ilustrasi oleh unDraw, ikon oleh Phosphor.</p>
	</div>
</footer>
