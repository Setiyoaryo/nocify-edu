<script lang="ts">
	// Mastery is always shown as number + word + bar, never colour alone (§37).
	let { value, label = '', compact = false }: { value: number | null; label?: string; compact?: boolean } = $props();
	const band = $derived(value === null ? null : value >= 70 ? 'strong' : value >= 40 ? 'developing' : 'needs');
	const word = { strong: 'Kuat', developing: 'Berkembang', needs: 'Perlu latihan' } as const;
	const fill = { strong: 'bg-leaf', developing: 'bg-sun', needs: 'bg-tangerine' } as const;
	const chip = { strong: 'bg-leaf-tint text-leaf-deep', developing: 'bg-sun-tint text-ink', needs: 'bg-tangerine-tint text-tangerine-deep' } as const;
</script>

<div>
	<div class="flex items-center justify-between gap-3">
		<span class="min-w-0 truncate font-semibold {compact ? 'text-sm' : ''}">{label}</span>
		<span class="flex shrink-0 items-center gap-2">
			{#if value === null}
				<span class="text-sm text-ink-soft">Belum ada data</span>
			{:else}
				{#if !compact}<span class="pill {chip[band!]}">{word[band!]}</span>{/if}
				<span class="w-11 text-right font-extrabold tabular-nums">{Math.round(value)}%</span>
			{/if}
		</span>
	</div>
	<div class="mt-2 h-2.5 overflow-hidden rounded-full bg-paper-deep" role="img" aria-label={value === null ? `${label}: belum ada data` : `${label}: ${Math.round(value)} persen, ${word[band!]}`}>
		{#if value !== null}<div class="bar-fill h-full rounded-full {fill[band!]}" style="width: {Math.max(3, value)}%"></div>{/if}
	</div>
</div>
