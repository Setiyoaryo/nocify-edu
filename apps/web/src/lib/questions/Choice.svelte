<script lang="ts">
	import type { Content, Response } from '$lib/types';
	import FigureGroup from '$lib/components/FigureGroup.svelte';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	let {
		content,
		response = $bindable(null),
		disabled = false,
		reveal = null,
		image = false
	}: { content: Content; response: Response | null; disabled?: boolean; reveal?: Response | null; image?: boolean } = $props();

	const name = `q-${Math.random().toString(36).slice(2)}`;
	let selected = $state<string | null>(null);
	$effect(() => {
		response = selected ? { option_id: selected } : null;
	});
	const letters = 'ABCDEF';
</script>

<fieldset class="grid gap-3 {image ? 'grid-cols-2' : 'grid-cols-1 sm:grid-cols-2'}" {disabled}>
	<legend class="sr-only">Pilih satu jawaban</legend>
	{#each content.options ?? [] as opt, i (opt.id)}
		{@const isAnswer = reveal?.option_id === opt.id}
		<label
			class="tile relative flex cursor-pointer items-center gap-3 px-4 py-3 text-left has-[:focus-visible]:outline-3 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-primary {image
				? 'min-h-36 flex-col justify-center text-center'
				: 'min-h-16'}"
			aria-checked={selected === opt.id}
			style={isAnswer ? 'border-color: var(--color-leaf-deep); box-shadow: 0 3px 0 var(--color-leaf-deep); background: var(--color-leaf-tint)' : ''}
		>
			<input class="sr-only" type="radio" {name} value={opt.id} bind:group={selected} />
			{#if !image}
				<span class="grid size-9 shrink-0 place-items-center rounded-full text-sm font-extrabold {selected === opt.id ? 'bg-primary text-white' : 'bg-paper-deep text-ink-soft'}" aria-hidden="true">{letters[i]}</span>
			{/if}
			{#if opt.image}
				<img src={opt.image} alt={opt.alt} class="h-20 w-auto" loading="lazy" />
			{:else if opt.figures?.length}
				<FigureGroup figures={opt.figures} size={36} />
			{/if}
			{#if opt.label}
				<span class={opt.figures?.length || opt.image ? 'text-sm font-semibold text-ink-soft' : 'text-xl font-bold'}>{opt.label}</span>
			{/if}
			{#if isAnswer}
				<!-- Inline for text options so it never covers the answer; a corner badge on picture options. -->
				<span class="anim-pop {image ? 'absolute top-2 right-2' : 'ml-auto'} grid size-8 shrink-0 place-items-center rounded-full bg-leaf-deep text-white">
					<CheckCircle size={20} weight="fill" aria-hidden="true" /><span class="sr-only">Jawaban benar</span>
				</span>
			{/if}
		</label>
	{/each}
</fieldset>
