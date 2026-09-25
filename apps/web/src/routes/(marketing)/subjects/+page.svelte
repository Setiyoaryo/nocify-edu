<script lang="ts">
	import Meta from '$lib/components/Meta.svelte';
	import PageHero from '$lib/components/PageHero.svelte';
	import CtaBand from '$lib/components/CtaBand.svelte';
	import { reveal } from '$lib/motion';
	import reading from '$lib/illustrations/reading.svg';
	import MathOperations from 'phosphor-svelte/lib/MathOperations';
	import BookOpenText from 'phosphor-svelte/lib/BookOpenText';
	import Plant from 'phosphor-svelte/lib/Plant';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	import type { Component } from 'svelte';

	type Subject = { name: string; icon: Component<any>; tone: string; color: string; chip: string; groups: [string, string[]][] };
	const subjects: Subject[] = [
		{
			name: 'Matematika', icon: MathOperations, tone: 'pastel-grape', color: '#5b5ce2', chip: 'bg-primary-tint text-primary-deep',
			groups: [
				['TK B', ['Membilang 1–10', 'Lebih banyak & lebih sedikit', 'Bentuk dasar', 'Melanjutkan pola']],
				['Kelas 1', ['Penjumlahan 1–20', 'Pengurangan 1–20', 'Soal cerita sederhana']],
				['Kelas 4', ['Perkalian', 'Pembagian', 'Soal cerita', 'Pecahan sederhana', 'Nilai tempat']]
			]
		},
		{
			name: 'Bahasa Indonesia', icon: BookOpenText, tone: 'pastel-peach', color: '#c9501c', chip: 'bg-tangerine-tint text-tangerine-deep',
			groups: [
				['TK B', ['Huruf awal', 'Urutan kejadian']],
				['Kelas 1', ['Suku kata', 'Kosakata benda']]
			]
		},
		{
			name: 'IPAS', icon: Plant, tone: 'pastel-mint', color: '#15804a', chip: 'bg-leaf-tint text-leaf-deep',
			groups: [['Kelas 3', ['Bagian tumbuhan', 'Wujud benda']]]
		}
	];
	const count = (s: Subject) => s.groups.reduce((n, [, k]) => n + k.length, 0);
	const grades = (s: Subject) => s.groups.map(([g]) => g).join(', ');
</script>

<Meta title="Pelajaran" description="Matematika, Bahasa Indonesia, dan IPAS untuk TK dan SD, disusun per kelas, bab, dan skill." />
<PageHero title="Pelajaran" lead="Setiap pelajaran dipecah menjadi bab dan skill, jadi perkembangan anak terlihat sampai ke kemampuan yang spesifik. Kelas lain menyusul bertahap." image={reading}>
	<ul class="anim-rise mt-6 flex flex-wrap gap-2" style="--d:180ms">
		{#each subjects as s}
			<li class="pill bg-white/15 py-1.5 text-sm text-white"><s.icon size={16} weight="fill" aria-hidden="true" />{s.name}</li>
		{/each}
	</ul>
</PageHero>

<div class="mx-auto max-w-6xl space-y-6 px-4 sm:px-6">
	{#each subjects as s}
		<section class="reveal {s.tone} relative overflow-hidden rounded-[1.8rem] p-5 sm:p-8" use:reveal aria-labelledby="sub-{s.name}">
			<span class="pointer-events-none absolute -top-10 -right-10 rotate-12 opacity-[0.1]" aria-hidden="true"><s.icon size={220} weight="fill" color={s.color} /></span>
			<div class="relative flex flex-wrap items-center gap-4">
				<span class="grid size-14 shrink-0 place-items-center rounded-full bg-white shadow-sm"><s.icon size={30} weight="duotone" color={s.color} aria-hidden="true" /></span>
				<div class="min-w-0 flex-1">
					<h2 id="sub-{s.name}" class="text-2xl sm:text-3xl">{s.name}</h2>
					<p class="text-sm font-semibold text-ink-soft">{count(s)} skill untuk {grades(s)}</p>
				</div>
				<a href="/register" class="pill-action hidden sm:inline-flex">Coba gratis</a>
			</div>
			<div class="relative mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each s.groups as [grade, skills]}
					<div class="rounded-2xl bg-white p-5 shadow-[var(--shadow-card)]">
						<span class="pill {s.chip}">{grade}</span>
						<ul class="mt-3 divide-y divide-line">
							{#each skills as k}
								<li class="flex items-center gap-2.5 py-2.5 font-semibold">
									<CheckCircle size={20} weight="fill" color={s.color} class="shrink-0" aria-hidden="true" />{k}
								</li>
							{/each}
						</ul>
					</div>
				{/each}
			</div>
		</section>
	{/each}
</div>

<CtaBand title="Coba challenge pertama hari ini" body="Pilih kelas anak Anda, dan soalnya akan menyesuaikan kemampuannya. Gratis untuk 1 anak." />
