<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	let { children } = $props();

	onMount(async () => {
		if (await session.load()) return;
		const next = encodeURIComponent(page.url.pathname + page.url.search);
		// Staff sign in through the unlisted /masuk-pengelola page, so the admin
		// area does not advertise the public parent/child login as its entry.
		const target = page.url.pathname.startsWith('/app/admin') ? '/masuk-pengelola' : '/login';
		goto(`${target}?next=${next}`, { replaceState: true });
	});
</script>

<svelte:head><meta name="robots" content="noindex" /></svelte:head>

{#if session.user}
	{@render children()}
{:else}
	<p class="grid min-h-dvh place-items-center text-ink-soft" aria-live="polite">Memuat…</p>
{/if}
