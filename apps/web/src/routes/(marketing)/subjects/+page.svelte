<script lang="ts">
	import Meta from '$lib/components/Meta.svelte';
	import PageHero from '$lib/components/PageHero.svelte';
	import reading from '$lib/illustrations/reading.svg';
	import MathOperations from 'phosphor-svelte/lib/MathOperations';
	import BookOpenText from 'phosphor-svelte/lib/BookOpenText';
	import Plant from 'phosphor-svelte/lib/Plant';
	const subjects = [
		{ name: 'Matematika', icon: MathOperations, tint: 'bg-primary-tint text-primary-deep', groups: [['TK B', ['Membilang 1–10', 'Lebih banyak & lebih sedikit', 'Bentuk dasar', 'Melanjutkan pola']], ['Kelas 1', ['Penjumlahan 1–20', 'Pengurangan 1–20', 'Soal cerita sederhana']], ['Kelas 4', ['Perkalian', 'Pembagian', 'Soal cerita', 'Pecahan sederhana', 'Nilai tempat']]] },
		{ name: 'Bahasa Indonesia', icon: BookOpenText, tint: 'bg-tangerine-tint text-tangerine-deep', groups: [['TK B', ['Huruf awal', 'Urutan kejadian']], ['Kelas 1', ['Suku kata', 'Kosakata benda']]] },
		{ name: 'IPAS', icon: Plant, tint: 'bg-leaf-tint text-leaf-deep', groups: [['Kelas 3', ['Bagian tumbuhan', 'Wujud benda']]] }
	] as const;
</script>

<Meta title="Pelajaran" description="Matematika, Bahasa Indonesia, dan IPAS untuk TK dan SD, disusun per kelas, bab, dan skill." />
<PageHero title="Pelajaran" lead="Setiap pelajaran dipecah menjadi bab dan skill, jadi perkembangan anak terlihat sampai ke kemampuan yang spesifik. Kelas lain menyusul bertahap." image={reading} />

<div class="mx-auto max-w-6xl space-y-6 px-4 sm:px-6">
	{#each subjects as s}
		<section class="card grid gap-6 p-6 sm:p-8 md:grid-cols-[14rem_1fr]">
			<div class="flex items-center gap-3 md:block">
				<span class="grid size-14 place-items-center rounded-2xl {s.tint}"><s.icon size={30} weight="duotone" aria-hidden="true" /></span>
				<h2 class="text-2xl md:mt-4">{s.name}</h2>
			</div>
			<div class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
				{#each s.groups as [grade, skills]}
					<div>
						<p class="text-sm font-bold text-ink-soft">{grade}</p>
						<ul class="mt-2 flex flex-wrap gap-2">
							{#each skills as k}<li class="rounded-full bg-paper-deep px-3 py-1.5 text-sm font-semibold">{k}</li>{/each}
						</ul>
					</div>
				{/each}
			</div>
		</section>
	{/each}
	<p class="pt-4"><a href="/register" class="btn btn-sun">Coba challenge pertama</a></p>
</div>
