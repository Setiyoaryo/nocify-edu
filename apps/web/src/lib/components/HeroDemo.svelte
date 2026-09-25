<script lang="ts">
	// A real, playable two-question challenge. It runs locally (no account, no
	// data sent) and fills in the parent's report as the visitor plays — the
	// product's core promise, shown rather than described (§4.3).
	import FigureGroup from './FigureGroup.svelte';
	import Burst from './Burst.svelte';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	import ArrowCounterClockwise from 'phosphor-svelte/lib/ArrowCounterClockwise';
	import Star from 'phosphor-svelte/lib/Star';
	import type { Fig } from '$lib/types';
	type Q = { skill: string; prompt: string; stimulus?: Fig[]; options: { id: string; figures?: Fig[]; label: string }[]; answer: string };
	const qs: Q[] = [
		{
			skill: 'Lebih banyak & lebih sedikit',
			prompt: 'Mana yang jeruknya lebih banyak?',
			options: [
				{ id: 'a', figures: [{ icon: 'orange', count: 3 }], label: '3 jeruk' },
				{ id: 'b', figures: [{ icon: 'orange', count: 5 }], label: '5 jeruk' }
			],
			answer: 'b'
		},
		{
			skill: 'Membilang 1–10',
			prompt: 'Ada berapa bintang?',
			stimulus: [{ icon: 'star', count: 7 }],
			options: [
				{ id: 'a', label: '6' },
				{ id: 'b', label: '7' },
				{ id: 'c', label: '8' }
			],
			answer: 'b'
		}
	];
	let step = $state(0);
	let tries = $state(0);
	let feedback = $state<{ ok: boolean; text: string } | null>(null);
	let results = $state<{ skill: string; first: boolean }[]>([]);
	const q = $derived(qs[Math.min(step, qs.length - 1)]);
	const finished = $derived(results.length === qs.length);
	const stars = $derived(results.filter((r) => r.first).length + 1);

	function choose(id: string) {
		if (feedback?.ok || finished) return;
		tries++;
		if (id === q.answer) {
			feedback = { ok: true, text: tries === 1 ? 'Hebat! Jawabanmu benar.' : 'Bagus! Kamu berhasil setelah mencoba lagi.' };
			results = [...results, { skill: q.skill, first: tries === 1 }];
		} else {
			feedback = { ok: false, text: 'Hampir benar. Coba hitung satu per satu lagi.' };
		}
	}
	function next() {
		step++;
		tries = 0;
		feedback = null;
	}
	function restart() {
		step = 0;
		tries = 0;
		feedback = null;
		results = [];
	}
</script>

<div class="grid gap-5 lg:grid-cols-[minmax(0,1.25fr)_minmax(0,1fr)]">
	<section aria-label="Contoh tampilan anak" class="card p-5 sm:p-7">
		<div class="mb-5 flex items-center justify-between">
			<span class="pill bg-primary-tint text-primary-deep">Tampilan anak</span>
			<span class="flex gap-1.5" aria-label="Soal {Math.min(step + 1, qs.length)} dari {qs.length}">
				{#each qs as _, i}<span class="h-2 w-8 rounded-full {i < results.length ? 'bg-leaf' : i === step ? 'bg-primary' : 'bg-line'}"></span>{/each}
			</span>
		</div>
		{#if !finished}
			{#key step}<p class="anim-rise text-2xl font-extrabold">{q.prompt}</p>{/key}
			{#if q.stimulus}<div class="mt-4 rounded-2xl bg-paper-deep py-4"><FigureGroup figures={q.stimulus} size={36} /></div>{/if}
			<div class="mt-4 grid gap-3 {q.options.length === 2 ? 'grid-cols-2' : 'grid-cols-3'}">
				{#each q.options as o}
					<button class="tile press flex min-h-24 flex-col items-center justify-center gap-2 px-2" onclick={() => choose(o.id)} disabled={feedback?.ok} aria-label={o.label}>
						{#if o.figures}<FigureGroup figures={o.figures} size={30} />{/if}
						<span class={o.figures ? 'text-sm font-semibold text-ink-soft' : 'text-3xl font-extrabold'}>{o.label}</span>
					</button>
				{/each}
			</div>
			<div class="mt-4 min-h-12" aria-live="polite">
				{#if feedback}
					{#key tries}
					<div class="flex items-center justify-between gap-3 rounded-2xl px-4 py-3 {feedback.ok ? 'bg-leaf-tint anim-rise' : 'bg-sun-tint anim-nudge'}">
						<p class="flex items-center gap-2 font-bold {feedback.ok ? 'text-leaf-deep' : 'text-ink'}">
							{#if feedback.ok}<span class="anim-pop relative inline-grid"><CheckCircle size={22} weight="fill" aria-hidden="true" /><Burst count={10} spread={46} /></span>{:else}<ArrowCounterClockwise size={22} weight="bold" aria-hidden="true" />{/if}
							{feedback.text}
						</p>
						{#if feedback.ok}<button class="btn btn-primary btn-sm" onclick={next}>Lanjut</button>{/if}
					</div>
					{/key}
				{/if}
			</div>
		{:else}
			<div class="anim-rise py-4 text-center">
				<p class="text-2xl font-extrabold">Challenge selesai!</p>
				<p class="relative mt-3 flex justify-center gap-1 text-sun-deep" aria-label="{stars} dari 3 bintang">
					{#each [1, 2, 3] as n}<span class="anim-pop" style="--d:{n * 140}ms"><Star size={40} weight="fill" class={n <= stars ? '' : 'text-line'} aria-hidden="true" /></span>{/each}
					<Burst spread={90} />
				</p>
				<button class="btn btn-quiet btn-sm mt-5" onclick={restart}>Main lagi</button>
			</div>
		{/if}
	</section>

	<section aria-label="Contoh laporan orang tua" class="rounded-[1.25rem] bg-night p-5 text-white sm:p-7">
		<span class="pill bg-white/10 text-white">Yang Anda lihat</span>
		<p class="mt-4 text-xl font-extrabold">Laporan Adi</p>
		<ul class="mt-4 space-y-3" aria-live="polite">
			{#each qs as item}
				{@const r = results.find((x) => x.skill === item.skill)}
				<li class="rounded-2xl p-3.5 transition-colors duration-500 {r ? 'bg-white/12' : 'bg-white/5'}">
					<p class="font-bold">{item.skill}</p>
					<p class="mt-0.5 flex items-center gap-1.5 text-sm {r ? (r.first ? 'text-leaf' : 'text-sun') : 'text-white/60'}">
						{#if r}
							<span class="anim-pop inline-flex items-center gap-1.5">{#if r.first}<CheckCircle size={16} weight="fill" aria-hidden="true" />Benar di percobaan pertama{:else}<ArrowCounterClockwise size={16} weight="bold" aria-hidden="true" />Benar setelah petunjuk{/if}</span>
						{:else}Menunggu jawaban…{/if}
					</p>
				</li>
			{/each}
		</ul>
		{#if finished}
			<p class="anim-rise mt-4 rounded-2xl bg-sun p-3.5 text-sm font-semibold text-ink" style="--d:250ms">
				<b>Saran:</b> {results.every((r) => r.first) ? 'Siap naik ke Membilang 11–20.' : 'Latih 3 challenge membilang minggu ini.'}
			</p>
		{/if}
	</section>
</div>
