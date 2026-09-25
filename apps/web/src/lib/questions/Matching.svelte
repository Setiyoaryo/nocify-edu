<script lang="ts">
	import type { Content, Response, Item } from '$lib/types';
	import Figure from '$lib/components/Figure.svelte';
	import { figure } from '$lib/figures';
	let {
		content,
		response = $bindable(null),
		disabled = false
	}: { content: Content; response: Response | null; disabled?: boolean } = $props();

	// Tap a left item, then its partner on the right. Pairs are marked with a
	// shared number, so matching never relies on colour alone (§24).
	let pairs = $state<Record<string, string>>({});
	let active = $state<string | null>(null);
	const left = $derived(content.left ?? []);
	const right = $derived(content.right ?? []);
	const pairNo = (leftId: string) => left.findIndex((l) => l.id === leftId) + 1;
	const leftOf = (rightId: string) => Object.keys(pairs).find((k) => pairs[k] === rightId);

	$effect(() => {
		response = Object.keys(pairs).length === left.length && left.length > 0 ? { pairs: { ...pairs } } : null;
	});

	function pickLeft(id: string) {
		if (pairs[id]) {
			const { [id]: _, ...rest } = pairs;
			pairs = rest;
		}
		active = active === id ? null : id;
	}
	function pickRight(id: string) {
		const owner = leftOf(id);
		if (owner) {
			const { [owner]: _, ...rest } = pairs;
			pairs = rest;
		}
		if (active) {
			pairs = { ...pairs, [active]: id };
			active = null;
		}
	}
	const name = (it: Item) => it.label || (it.icon ? figure(it.icon).label : '');
</script>

{#snippet face(it: Item)}
	{#if it.icon}<Figure name={it.icon} color={it.color} size={34} />{/if}
	{#if it.label}<span>{it.label}</span>{/if}
{/snippet}

<p class="mb-3 text-center font-semibold text-ink-soft" aria-live="polite">
	{active ? 'Sekarang ketuk pasangannya di kanan' : 'Ketuk satu di kiri, lalu pasangannya di kanan'}
</p>
<div class="grid grid-cols-2 gap-3 sm:gap-6">
	<div class="space-y-2">
		{#each left as l (l.id)}
			<button type="button" class="tile relative flex min-h-16 w-full items-center justify-center gap-2 px-3 text-lg" {disabled}
				aria-label={name(l)} aria-pressed={active === l.id || !!pairs[l.id]} onclick={() => pickLeft(l.id)}>
				{@render face(l)}
				{#if pairs[l.id]}<span class="anim-pop absolute -top-2 -left-2 grid size-7 place-items-center rounded-full bg-primary text-sm font-extrabold text-white">{pairNo(l.id)}</span>{/if}
			</button>
		{/each}
	</div>
	<div class="space-y-2">
		{#each right as r (r.id)}
			{@const owner = leftOf(r.id)}
			<button type="button" class="tile relative flex min-h-16 w-full items-center justify-center gap-2 px-3 text-lg" {disabled}
				aria-label={name(r)} aria-pressed={!!owner} onclick={() => pickRight(r.id)}>
				{@render face(r)}
				{#if owner}<span class="anim-pop absolute -top-2 -right-2 grid size-7 place-items-center rounded-full bg-primary text-sm font-extrabold text-white">{pairNo(owner)}</span>{/if}
			</button>
		{/each}
	</div>
</div>
