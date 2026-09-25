<script lang="ts">
	import type { Content, Response } from '$lib/types';
	let {
		response = $bindable(null),
		disabled = false,
		onsubmit
	}: { content: Content; response: Response | null; disabled?: boolean; onsubmit?: () => void } = $props();
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
