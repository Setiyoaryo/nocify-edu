<script lang="ts">
	import { session } from '$lib/session.svelte';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	onMount(() => goto(session.isStaff ? '/app/admin' : '/app/parent', { replaceState: true }));
</script>
