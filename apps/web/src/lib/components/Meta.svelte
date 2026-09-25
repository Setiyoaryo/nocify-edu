<script lang="ts">
	import { page } from '$app/state';
	let { title, description, index = true }: { title: string; description: string; index?: boolean } = $props();
	const full = $derived(title === 'EduPlay' ? title : `${title} | EduPlay`);
</script>

<svelte:head>
	<title>{full}</title>
	<meta name="description" content={description} />
	{#if !index}<meta name="robots" content="noindex" />{/if}
	<meta property="og:type" content="website" />
	<meta property="og:site_name" content="EduPlay" />
	<meta property="og:title" content={full} />
	<meta property="og:description" content={description} />
	<meta property="og:url" content={page.url.origin + page.url.pathname} />
	<link rel="canonical" href={page.url.origin + page.url.pathname} />
</svelte:head>
