<script lang="ts">
	import type { Component } from 'svelte';
	let { label, value, icon, tint = 'bg-primary-tint text-primary-deep' }: { label: string; value: string | number; icon: Component<any>; tint?: string } = $props();
	const Icon = $derived(icon);
</script>

<div class="lift flex items-center gap-3 rounded-2xl bg-paper-deep/70 p-3.5">
	<span class="grid size-10 shrink-0 place-items-center rounded-xl {tint}"><Icon size={22} weight="duotone" aria-hidden="true" /></span>
	<div class="min-w-0">
		<p class="text-xs leading-snug font-semibold text-ink-soft">{label}</p>
		<p class="text-xl leading-tight font-extrabold tabular-nums">{value}</p>
	</div>
</div>
