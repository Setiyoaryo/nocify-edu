<script lang="ts">
	import type { Content, Response, Item } from '$lib/types';
	import Figure from '$lib/components/Figure.svelte';
	import X from 'phosphor-svelte/lib/X';
	let {
		content,
		response = $bindable(null),
		disabled = false,
		reveal = null
	}: { content: Content; response: Response | null; disabled?: boolean; reveal?: any } = $props();

	// Tap-to-order instead of drag: far more reliable for small fingers on
	// touchscreens (§31.2) and fully keyboard/screen-reader operable.
	let placed = $state<string[]>([]);
	const items = $derived(content.items ?? []);
	const byId = (id: string) => items.find((i) => i.id === id) as Item;
	$effect(() => {
		response = placed.length === items.length && items.length > 0 ? { order: [...placed] } : null;
	});
</script>

{#snippet chip(it: Item)}
	{#if it.icon}<Figure name={it.icon} color={it.color} size={30} />{/if}
	<span>{it.label}</span>
{/snippet}

<div class="space-y-5">
	<ol class="space-y-2" aria-label="Urutanmu">
		{#each items as _, i}
			{@const id = placed[i]}
			<li class="flex items-center gap-3">
				<span class="grid size-9 shrink-0 place-items-center rounded-full bg-primary-tint text-lg font-extrabold text-primary-deep">{i + 1}</span>
				{#if id}
					<button type="button" class="tile anim-rise flex min-h-14 flex-1 items-center gap-3 px-4 text-left text-lg" style="--d:0ms" {disabled}
						onclick={() => (placed = placed.filter((p) => p !== id))} aria-label="{byId(id).label}, ketuk untuk melepas">
						{@render chip(byId(id))}<X class="ml-auto shrink-0 text-ink-soft" size={18} weight="bold" aria-hidden="true" />
					</button>
				{:else}
					<div class="flex min-h-14 flex-1 items-center rounded-2xl border-2 border-dashed border-line px-4 text-sm font-semibold text-ink-soft">
						{i === placed.length ? 'Ketuk pilihan di bawah' : ''}
					</div>
				{/if}
			</li>
		{/each}
	</ol>
	{#if reveal?.order?.length}
		<div class="anim-rise rounded-2xl bg-leaf-tint p-3 text-center text-sm font-bold text-leaf-deep">
			Urutan yang benar: <span class="rounded-lg bg-white px-2 py-0.5">{reveal.order.map((id: string) => byId(id)?.label ?? id).join(' → ')}</span>
		</div>
	{/if}
	<div class="flex flex-wrap justify-center gap-2" role="group" aria-label="Pilihan">
		{#each items.filter((it) => !placed.includes(it.id)) as it (it.id)}
			<button type="button" class="tile flex min-h-14 items-center gap-2 px-4 text-lg" {disabled} onclick={() => (placed = [...placed, it.id])}>{@render chip(it)}</button>
		{/each}
	</div>
</div>
