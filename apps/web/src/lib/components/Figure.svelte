<script lang="ts">
	import { figure, palette } from '$lib/figures';
	// One pictured object. Decorative unless given a label.
	let { name, color = '', size = 40, label = '', weight = 'fill' }: { name: string; color?: string; size?: number; label?: string; weight?: 'fill' | 'duotone' | 'bold' } = $props();
	const f = $derived(figure(name));
</script>

<span class="inline-grid place-items-center" role={label ? 'img' : undefined} aria-label={label || undefined} aria-hidden={label ? undefined : 'true'}>
	<f.icon size={size} weight={weight} color={palette[color || f.color]} aria-hidden="true" />
</span>
