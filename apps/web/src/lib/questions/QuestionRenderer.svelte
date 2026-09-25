<script lang="ts" module>
	import type { Component } from 'svelte';
	import Choice from './Choice.svelte';
	import TapCount from './TapCount.svelte';
	import NumberInput from './NumberInput.svelte';
	import ShortAnswer from './ShortAnswer.svelte';
	import Sequence from './Sequence.svelte';
	import Matching from './Matching.svelte';

	// Registry: a new question type is one component + one backend grader (§9, §15).
	export const renderers: Record<string, { component: Component<any>; props?: Record<string, unknown> }> = {
		single_choice: { component: Choice },
		image_choice: { component: Choice, props: { image: true } },
		tap_count: { component: TapCount },
		number_input: { component: NumberInput },
		short_answer: { component: ShortAnswer },
		sequence: { component: Sequence },
		matching: { component: Matching }
	};
</script>

<script lang="ts">
	import type { Content, Response } from '$lib/types';
	import { speech } from '$lib/speech.svelte';
	import FigureGroup from '$lib/components/FigureGroup.svelte';
	import SpeakerHigh from 'phosphor-svelte/lib/SpeakerHigh';
	import SpeakerSimpleHigh from 'phosphor-svelte/lib/SpeakerSimpleHigh';

	let {
		type,
		content,
		response = $bindable(null),
		disabled = false,
		reveal = null,
		band = 'lower',
		onsubmit
	}: {
		type: string;
		content: Content;
		response: Response | null;
		disabled?: boolean;
		reveal?: Response | null;
		band?: string;
		onsubmit?: () => void;
	} = $props();

	const r = $derived(renderers[type]);
	const promptSize = $derived(band === 'early' ? 'text-2xl sm:text-[2.1rem]' : band === 'lower' ? 'text-2xl sm:text-3xl' : 'text-xl sm:text-2xl');
	const spoken = $derived(content.audio_text || content.prompt);
</script>

<!-- Mobile: prompt, stimulus, interaction stacked. Desktop: prompt + stimulus beside the interaction (§31.4). -->
<div class="grid gap-6 md:grid-cols-[minmax(0,5fr)_minmax(0,7fr)] md:items-start md:gap-10">
	<div class="space-y-4">
		<div class="flex items-start gap-3">
			<h2 class="flex-1 {promptSize} leading-snug font-extrabold">{content.prompt}</h2>
			{#if speech.supported}
				<button
					type="button"
					class="tile grid size-14 shrink-0 place-items-center text-primary"
					aria-label={speech.muted ? 'Suara dimatikan' : 'Dengarkan soal'}
					disabled={speech.muted}
					onclick={() => speech.say(spoken)}
				>
					{#if speech.speaking}<SpeakerHigh size={26} weight="fill" aria-hidden="true" />{:else}<SpeakerSimpleHigh size={26} weight="bold" aria-hidden="true" />{/if}
				</button>
			{/if}
		</div>
		{#if content.figures?.length}
			<div class="rounded-3xl bg-paper-deep px-4 py-6">
				<FigureGroup figures={content.figures} size={band === 'early' ? 52 : 44} />
			</div>
		{/if}
		{#if content.image}
			<img src={content.image} alt={content.image_alt} class="mx-auto max-h-56 w-auto" />
		{/if}
	</div>
	<div>
		{#if r}
			<r.component {content} bind:response {disabled} {reveal} {onsubmit} {...r.props} />
		{:else}
			<p class="text-error">Tipe soal "{type}" belum didukung.</p>
		{/if}
	</div>
</div>
