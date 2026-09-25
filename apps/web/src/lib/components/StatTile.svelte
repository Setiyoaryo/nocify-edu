<script lang="ts">
	import type { Component } from 'svelte';
	// Stacks icon above the text so three tiles still fit side by side on a phone.
	let { label, value, icon, tint = 'bg-primary-tint text-primary-deep' }: { label: string; value: string | number; icon: Component<any>; tint?: string } = $props();
	const Icon = $derived(icon);
</script>

<div class="lift flex h-full flex-col gap-2 rounded-2xl bg-paper-deep/70 p-3 sm:p-3.5 xl:flex-row xl:items-center xl:gap-3">
	<span class="grid size-9 shrink-0 place-items-center rounded-xl xl:size-10 {tint}"><Icon size={20} weight="duotone" aria-hidden="true" /></span>
	<div class="min-w-0">
		<p class="text-xs leading-snug font-semibold text-ink-soft">{label}</p>
		<p class="text-xl leading-tight font-extrabold tabular-nums">{value}</p>
	</div>
</div>
