<script lang="ts">
	import { get, patch } from '$lib/api';
	import { page } from '$app/state';
	import { getContext } from 'svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import LockSimple from 'phosphor-svelte/lib/LockSimple';
	const id = page.params.childId as string;
	const ctx: any = getContext('play');
	let avatars = $state<any[]>([]);
	get(`/children/${id}/rewards`).then((r) => (avatars = r.items.filter((i: any) => i.kind === 'avatar')));
	async function choose(code: string) {
		await patch(`/children/${id}`, { avatar: code });
		await ctx.refresh();
	}
</script>

<svelte:head><title>Profil | EduPlay</title></svelte:head>

<h1 class="mt-3 text-3xl sm:text-4xl">Pilih avatarmu</h1>
<p class="mt-1 text-ink-soft">Kumpulkan XP untuk membuka avatar baru.</p>
<div class="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-4" role="radiogroup" aria-label="Avatar">
	{#each avatars as a (a.code)}
		<button role="radio" aria-checked={ctx.child?.avatar === a.code} class="tile relative flex flex-col items-center gap-2 py-5 disabled:opacity-45" disabled={!a.owned} onclick={() => choose(a.code)}>
			<Avatar code={a.icon} size={72} />
			<span class="font-extrabold">{a.name}</span>
			{#if !a.owned}<span class="flex items-center gap-1 text-xs font-semibold text-ink-soft"><LockSimple size={14} weight="fill" aria-hidden="true" />{a.xp_required} XP</span>{/if}
		</button>
	{/each}
</div>
