<script lang="ts">
	// Counts from 0 (or the previous value) to `value`: used for scores, XP, stars.
	import { Tween } from 'svelte/motion';
	import { cubicOut } from 'svelte/easing';
	import { prefersReducedMotion } from '$lib/motion';
	import { untrack } from 'svelte';
	let { value, duration = 900, delay = 0, suffix = '' }: { value: number; duration?: number; delay?: number; suffix?: string } = $props();
	// Timing is fixed at creation; only the target value is reactive.
	const t = untrack(() => new Tween(0, { duration: prefersReducedMotion() ? 0 : duration, easing: cubicOut, delay }));
	$effect(() => {
		t.target = value;
	});
</script>

<span class="tabular-nums">{Math.round(t.current)}{suffix}</span>
