<script lang="ts">
	// Closing call to action for secondary marketing pages: gives the sign-up
	// button context and space instead of leaving it stranded above the footer.
	import { reveal } from '$lib/motion';
	import Rocket from 'phosphor-svelte/lib/Rocket';
	let {
		title = 'Siap mencoba bersama anak Anda?',
		body = 'Gratis untuk 1 anak, tanpa kartu kredit. Challenge pertama bisa dimulai dalam dua menit.'
	}: { title?: string; body?: string } = $props();
</script>

<section class="mx-auto max-w-6xl px-4 pt-16 pb-16 sm:px-6 sm:pb-20">
	<div class="reveal relative overflow-hidden rounded-[1.8rem] bg-night p-7 text-white sm:p-10" use:reveal>
		<span class="anim-float pointer-events-none absolute -right-8 -bottom-10 opacity-15" aria-hidden="true"><Rocket size={210} weight="fill" /></span>
		<div class="relative grid items-center gap-6 lg:grid-cols-[1fr_auto]">
			<div>
				<h2 class="text-[1.6rem] sm:text-3xl">{title}</h2>
				<p class="mt-2 max-w-xl text-white/75">{body}</p>
			</div>
			<div class="grid gap-3 sm:flex">
				<a href="/register" class="btn btn-sun">Daftar gratis</a>
				<a href="/#coba" class="btn btn-ghost-light">Coba contoh soal</a>
			</div>
		</div>
	</div>
</section>
