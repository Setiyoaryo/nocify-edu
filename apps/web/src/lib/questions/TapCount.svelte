<script lang="ts">
	import type { Content, Response } from '$lib/types';
	import Figure from '$lib/components/Figure.svelte';
	import { figure } from '$lib/figures';
	let {
		content,
		response = $bindable(null),
		disabled = false,
		reveal = null
	}: { content: Content; response: Response | null; disabled?: boolean; reveal?: any } = $props();

	// Tapping each object is a counting aid (one-to-one correspondence); the
	// answer is the number the child then picks.
	let order = $state<number[]>([]);
	let picked = $state<number | null>(null);
	$effect(() => {
		response = picked === null ? null : { value: picked };
	});
	function tap(i: number) {
		order = order.includes(i) ? order.filter((x) => x !== i) : [...order, i];
	}
	const noun = $derived(figure(content.symbol ?? '').label);
</script>

<div class="space-y-5">
	<div class="flex flex-wrap justify-center gap-2 sm:gap-3" role="group" aria-label="Ketuk setiap {noun} untuk menghitung">
		{#each Array(content.count ?? 0) as _, i}
			{@const n = order.indexOf(i) + 1}
			<button
				type="button"
				class="tile relative grid size-16 place-items-center sm:size-20"
				aria-pressed={n > 0}
				aria-label={n > 0 ? `${noun} ke-${n}` : `${noun}, belum dihitung`}
				{disabled}
				onclick={() => tap(i)}
			>
				<Figure name={content.symbol ?? 'circle'} color={content.color} size={40} />
				{#if n > 0}
					<span class="anim-pop absolute -top-2 -right-2 grid size-7 place-items-center rounded-full bg-primary text-sm font-extrabold text-white">{n}</span>
				{/if}
			</button>
		{/each}
	</div>
	<p class="text-center font-semibold text-ink-soft" aria-live="polite">
		{order.length === 0 ? 'Ketuk untuk menghitung' : `Kamu sudah menghitung ${order.length}`}
	</p>
	<div class="flex justify-center gap-3" role="radiogroup" aria-label="Pilih jumlahnya">
		{#each content.choices ?? [] as c}
			{@const isAnswer = reveal?.value === c}
			<button
				type="button"
				role="radio"
				class="tile size-20 text-3xl font-extrabold"
				aria-checked={picked === c}
				style={isAnswer ? 'border-color: var(--color-leaf-deep); box-shadow: 0 3px 0 var(--color-leaf-deep); background: var(--color-leaf-tint)' : ''}
				{disabled}
				onclick={() => (picked = c)}
			>{c}</button>
		{/each}
	</div>
	{#if reveal?.value !== undefined}
		<p class="anim-rise text-center text-sm font-bold text-leaf-deep">
			Jawaban benar: <span class="rounded-lg bg-leaf-tint px-2 py-0.5">{reveal.value}</span>
		</p>
	{/if}
</div>
