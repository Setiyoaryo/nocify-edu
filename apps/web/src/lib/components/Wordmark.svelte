<script lang="ts">
	// Text wordmark with a simple geometric mark (brand review pending, §25).
	let { href = '/', light = false }: { href?: string; light?: boolean } = $props();
</script>

<a {href} class="inline-flex min-h-11 items-center gap-2 text-[1.45rem] font-extrabold tracking-tight {light ? 'text-white' : 'text-ink'}">
	<svg width="30" height="30" viewBox="0 0 32 32" aria-hidden="true">
		<rect width="32" height="32" rx="9" fill={light ? '#fff' : '#5b5ce2'} />
		<path d="M9 17.5c0-4.4 3.1-7.5 7.2-7.5 4 0 6.8 2.9 6.8 7v1.2H13.2c.4 2 2 3.2 4.2 3.2 1.4 0 2.6-.5 3.6-1.3l1.6 2.3c-1.4 1.2-3.3 1.9-5.4 1.9C12.3 24.3 9 21.6 9 17.5Zm4.3-1.7h5.6c-.3-1.8-1.4-2.9-2.8-2.9-1.5 0-2.5 1.1-2.8 2.9Z" fill={light ? '#5b5ce2' : '#fff'} />
		<circle cx="25" cy="8" r="3" fill="#ffc857" />
	</svg>
	<span>edu<span class={light ? 'text-sun' : 'text-primary'}>play</span></span>
</a>
