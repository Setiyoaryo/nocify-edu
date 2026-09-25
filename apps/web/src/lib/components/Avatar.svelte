<script lang="ts">
	import Figure from './Figure.svelte';
	import { figure, palette } from '$lib/figures';
	// Child avatar: a Phosphor animal on a tinted disc. Codes come from rewards (kind = avatar).
	let { code, size = 48, ring = false }: { code: string; size?: number; ring?: boolean } = $props();
	const f = $derived(figure(code));
</script>

<span
	class="inline-grid shrink-0 place-items-center rounded-full {ring ? 'ring-4 ring-white' : ''}"
	style="width:{size}px;height:{size}px;background:{palette[f.color]}22"
	aria-hidden="true"
>
	<Figure name={code} size={Math.round(size * 0.62)} />
</span>
