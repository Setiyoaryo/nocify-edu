<script lang="ts">
	import type { Content, Response } from '$lib/types';
	import Backspace from 'phosphor-svelte/lib/Backspace';
	let {
		content,
		response = $bindable(null),
		disabled = false,
		onsubmit
	}: { content: Content; response: Response | null; disabled?: boolean; onsubmit?: () => void } = $props();

	// On-screen keypad keeps the CTA visible on phones (the native keyboard
	// would cover it, §33); physical keyboards can still type into the field.
	let text = $state('');
	$effect(() => {
		const v = Number(text.replace(',', '.'));
		response = text.trim() !== '' && text !== '-' && Number.isFinite(v) ? { value: v } : null;
	});
	const keys = ['1', '2', '3', '4', '5', '6', '7', '8', '9', '-', '0', 'hapus'];
	function press(k: string) {
		if (k === 'hapus') text = text.slice(0, -1);
		else if (k === '-') text = text.startsWith('-') ? text.slice(1) : '-' + text;
		else if (text.replace('-', '').length < 6) text += k;
	}
</script>

<div class="mx-auto max-w-xs space-y-3">
	<label class="block">
		<span class="sr-only">Jawabanmu</span>
		<input
			class="field h-16 text-center text-4xl font-extrabold"
			inputmode="none"
			autocomplete="off"
			bind:value={text}
			{disabled}
			placeholder="?"
			onkeydown={(e) => e.key === 'Enter' && response && onsubmit?.()}
			oninput={() => (text = text.replace(/[^0-9,.-]/g, ''))}
		/>
		{#if content.unit}<span class="mt-1 block text-center text-ink-soft">{content.unit}</span>{/if}
	</label>
	<div class="grid grid-cols-3 gap-2">
		{#each keys as k}
			<button type="button" class="tile grid h-14 place-items-center text-2xl" {disabled} onclick={() => press(k)} aria-label={k === 'hapus' ? 'Hapus' : k === '-' ? 'Tanda minus' : k}>
				{#if k === 'hapus'}<Backspace size={26} weight="bold" aria-hidden="true" />{:else}{k}{/if}
			</button>
		{/each}
	</div>
</div>
