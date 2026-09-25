<script lang="ts">
	// Sticky sign-up bar on phones, shown once the visitor scrolls past the hero.
	import { page } from '$app/state';
	let y = $state(0);
	const hidden = $derived(['/register', '/login'].includes(page.url.pathname));
</script>

<svelte:window bind:scrollY={y} />

{#if !hidden}
	<div class="safe-bottom fixed inset-x-0 bottom-0 z-30 border-t border-line bg-paper/95 px-4 pt-3 backdrop-blur transition-transform duration-300 md:hidden {y > 520 ? 'translate-y-0' : 'translate-y-full'}"
		aria-hidden={y <= 520}>
		<div class="flex items-center gap-3">
			<p class="flex-1 text-sm leading-tight font-bold">Gratis untuk 1 anak.<span class="block font-semibold text-ink-soft">Tanpa kartu kredit.</span></p>
			<a href="/register" class="btn btn-sun" tabindex={y > 520 ? 0 : -1}>Daftar gratis</a>
		</div>
	</div>
{/if}
