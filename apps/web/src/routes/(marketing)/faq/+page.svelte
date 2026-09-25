<script lang="ts">
	import Meta from '$lib/components/Meta.svelte';
	import PageHero from '$lib/components/PageHero.svelte';
	import CtaBand from '$lib/components/CtaBand.svelte';
	import { faqs } from '$lib/content/faq';
	import Plus from 'phosphor-svelte/lib/Plus';
	const ld = JSON.stringify({
		'@context': 'https://schema.org',
		'@type': 'FAQPage',
		mainEntity: faqs.map(([q, a]) => ({ '@type': 'Question', name: q, acceptedAnswer: { '@type': 'Answer', text: a } }))
	});
</script>

<svelte:head>
	{@html '<script type="application/ld+json">' + ld + '</' + 'script>'}
</svelte:head>
<Meta title="Pertanyaan umum" description="Jawaban atas pertanyaan orang tua tentang EduPlay: usia, durasi, pembayaran, langganan, dan data anak." />
<PageHero title="Pertanyaan umum" lead="Tidak menemukan jawabannya? Hubungi kami dari area orang tua." />

<div class="mx-auto max-w-3xl space-y-3 px-4 sm:px-6">
	{#each faqs as [q, a]}
		<details class="card group">
			<summary class="flex min-h-14 cursor-pointer list-none items-center justify-between gap-4 px-5 py-4 text-lg font-bold">
				{q}<Plus size={20} weight="bold" class="shrink-0 text-primary transition-transform group-open:rotate-45" aria-hidden="true" />
			</summary>
			<p class="px-5 pb-5 text-ink-soft">{a}</p>
		</details>
	{/each}
</div>

<CtaBand title="Masih ada pertanyaan?" body="Cara terbaik untuk tahu adalah mencoba. Gratis untuk 1 anak, tanpa kartu kredit." />
