<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	let { children } = $props();

	onMount(async () => {
		if (!(await session.load())) goto('/login?next=' + encodeURIComponent(page.url.pathname + page.url.search), { replaceState: true });
	});
</script>

<svelte:head><meta name="robots" content="noindex" /></svelte:head>

{#if session.user}
	{@render children()}
{:else}
	<p class="grid min-h-dvh place-items-center text-ink-soft" aria-live="polite">Memuat…</p>
{/if}
