<script lang="ts">
	import type { Content, Response } from '$lib/types';
	let {
		response = $bindable(null),
		disabled = false,
		reveal = null,
		onsubmit
	}: { content: Content; response: Response | null; disabled?: boolean; reveal?: any; onsubmit?: () => void } = $props();
	let text = $state('');
	$effect(() => {
		response = text.trim() ? { text } : null;
	});
</script>

<label class="mx-auto block max-w-md">
	<span class="label">Tulis jawabanmu</span>
	<input
		class="field h-14 text-xl"
		bind:value={text}
		{disabled}
		autocomplete="off"
		autocapitalize="off"
		spellcheck="false"
		maxlength="80"
		onkeydown={(e) => e.key === 'Enter' && response && onsubmit?.()}
	/>
</label>
{#if reveal}
	{@const ans = reveal.accepted ? reveal.accepted.join(' / ') : reveal.text ?? ''}
	{#if ans}
		<p class="anim-rise mt-3 text-center text-sm font-bold text-leaf-deep">
			Jawaban benar: <span class="rounded-lg bg-leaf-tint px-2 py-0.5">{ans}</span>
		</p>
	{/if}
{/if}
