<script lang="ts">
	import { get, post, ApiError } from '$lib/api';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { getContext } from 'svelte';
	import QuestionRenderer from '$lib/questions/QuestionRenderer.svelte';
	import Figure from '$lib/components/Figure.svelte';
	import Burst from '$lib/components/Burst.svelte';
	import CountUp from '$lib/components/CountUp.svelte';
	import { speech, chime } from '$lib/speech.svelte';
	import winnerArt from '$lib/illustrations/winner.svg';
	import wellDone from '$lib/illustrations/well-done.svg';
	import X from 'phosphor-svelte/lib/X';
	import Lightbulb from 'phosphor-svelte/lib/Lightbulb';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	import ArrowCounterClockwise from 'phosphor-svelte/lib/ArrowCounterClockwise';
	import Info from 'phosphor-svelte/lib/Info';
	import Star from 'phosphor-svelte/lib/Star';
	import TrendUp from 'phosphor-svelte/lib/TrendUp';
	import Flag from 'phosphor-svelte/lib/Flag';
	import Lightning from 'phosphor-svelte/lib/Lightning';
	import type { Response } from '$lib/types';

	const { childId, assessmentId } = page.params as { childId: string; assessmentId: string };
	const ctx: any = getContext('play');

	type Phase = 'answer' | 'retry' | 'final' | 'mini' | 'finishing' | 'result';
	let asm = $state<any>(null);
	let phase = $state<Phase>('answer');
	let response = $state<Response | null>(null);
	let feedback = $state<{ ok: boolean; text: string } | null>(null);
	let hint = $state('');
	let explanation = $state('');
	let reveal = $state<Response | null>(null);
	let pending = $state<any>(null); // next assessment view, applied on "Lanjut"
	let done = $state(false);
	let mini = $state(false);
	let result = $state<any>(null);
	let error = $state('');
	let busy = $state(false);
	let shownAt = Date.now();
	let lastWrong = $state(''); // the retry button waits until the child changes the answer

	get(`/assessments/${assessmentId}`)
		.then((r) => {
			asm = r.assessment;
			if (asm.status === 'completed') goto(`/app/play/${childId}`, { replaceState: true });
			hint = asm.current?.hint ?? '';
			if (asm.current?.attempts > 0) phase = 'retry';
		})
		.catch((e) => (error = e.message));

	const item = $derived(asm?.current);
	const answeredCount = $derived(asm?.results.length ?? 0);

	async function askHint() {
		if (!item || hint) return;
		const r = await post(`/assessments/${assessmentId}/items/${item.id}/hint`);
		hint = r.hint;
		speech.say(r.hint);
	}

	async function submit() {
		if (!item || !response || busy || JSON.stringify(response) === lastWrong) return;
		busy = true;
		error = '';
		try {
			const r = await post(`/assessments/${assessmentId}/answers`, { item_id: item.id, response, response_time_ms: Date.now() - shownAt });
			feedback = { ok: r.correct, text: r.feedback };
			speech.say(r.feedback);
			if (r.correct) chime();
			if (!r.final) {
				phase = 'retry';
				hint = r.hint;
				lastWrong = JSON.stringify(response);
			} else {
				phase = 'final';
				explanation = r.explanation;
				reveal = r.correct_answer ?? null;
				pending = r.assessment;
				done = r.done;
				if (r.mini_reward) mini = true;
			}
		} catch (e) {
			error = e instanceof ApiError ? e.message : 'Jawaban belum terkirim. Coba lagi.';
		} finally {
			busy = false;
		}
	}

	async function next() {
		speech.stop();
		if (mini) {
			mini = false;
			phase = 'mini';
			return;
		}
		if (done) return finish();
		asm = pending;
		pending = null;
		phase = 'answer';
		feedback = null;
		hint = '';
		explanation = '';
		reveal = null;
		response = null;
		lastWrong = '';
		shownAt = Date.now();
		if (!asm.current) finish();
	}

	async function finish() {
		phase = 'finishing';
		try {
			result = await post(`/assessments/${assessmentId}/complete`);
			phase = 'result';
			await ctx.refresh();
		} catch (e) {
			error = e instanceof ApiError ? e.message : 'Gagal menyimpan hasil.';
			phase = 'final';
		}
	}

	const cheer = (score: number) => (score >= 90 ? 'Luar biasa!' : score >= 60 ? 'Kerja bagus!' : 'Kamu sudah berusaha, hebat!');
	const improved = $derived(result ? result.skills.filter((s: any) => s.after > (s.before ?? 0)) : []);
</script>

<svelte:head>
	<title>Challenge | EduPlay</title>
	<!-- Keep scrolled-to/focused controls clear of the sticky action bar (§33). -->
	<style>html { scroll-padding-bottom: 11rem; }</style>
</svelte:head>

{#if error && !asm}
	<p class="field-error mt-10 text-center" role="alert">{error}</p>
{:else if !asm}
	<p class="mt-10 text-center text-ink-soft">Menyiapkan challenge…</p>
{:else if phase === 'result' && result}
	<section class="mx-auto mt-4 max-w-lg" aria-live="polite">
		<div class="card anim-rise-scale overflow-hidden text-center">
			<div class="px-6 pt-8 pb-6 text-white" style="background: linear-gradient(140deg, #5b5ce2 0%, #7c5ce6 55%, #9b5ce0 100%)">
				<p class="font-semibold text-white/80">{asm.focus_skill ?? asm.subject_name}</p>
				<h1 class="anim-rise mt-1 text-4xl" style="--d:120ms">{cheer(result.result.score)}</h1>
				<p class="stars relative mt-5 flex justify-center gap-2" aria-label="{result.result.stars} dari 3 bintang">
					<Burst count={16} spread={120} />
					{#each [1, 2, 3] as n}<span style="--i:{n}"><Star size={64} weight="fill" class={n <= result.result.stars ? 'text-sun' : 'text-white/25'} aria-hidden="true" /></span>{/each}
				</p>
			</div>
			<img src={result.result.score >= 90 ? winnerArt : wellDone} alt="" class="anim-rise mx-auto w-44 pt-5" style="--d:500ms" />
			<div class="grid grid-cols-2 gap-3 px-6 pt-4 pb-6">
				<p class="rounded-2xl bg-leaf-tint p-3"><b class="block text-2xl text-leaf-deep"><CountUp value={result.result.correct} delay={600} />/{result.result.total}</b><span class="text-sm font-semibold">benar</span></p>
				<p class="rounded-2xl bg-primary-tint p-3"><b class="flex items-center justify-center gap-1 text-2xl text-primary-deep"><Lightning size={22} weight="fill" aria-hidden="true" />+<CountUp value={result.result.xp} delay={700} duration={1100} /></b><span class="text-sm font-semibold">XP</span></p>
			</div>
		</div>

		{#if result.mission}
			<div class="card anim-rise mt-4 flex items-center gap-4 p-5" style="--d:800ms">
				<span class="grid size-12 shrink-0 place-items-center rounded-2xl bg-tangerine-tint text-tangerine-deep"><Flag size={26} weight="fill" class="anim-wiggle" aria-hidden="true" /></span>
				<div class="flex-1">
					<p class="font-extrabold">{result.mission.status === 'completed' ? 'Misi selesai!' : 'Misi bertambah'}</p>
					<p class="text-sm text-ink-soft">{result.mission.skill_name}: {result.mission.completed} dari {result.mission.target} challenge</p>
				</div>
			</div>
		{/if}

		{#if improved.length}
			<div class="card anim-rise mt-4 p-5" style="--d:900ms">
				<p class="flex items-center gap-2 font-extrabold"><TrendUp size={22} weight="bold" class="text-leaf-deep" aria-hidden="true" />Kemampuanmu bertambah</p>
				<ul class="mt-2 flex flex-wrap gap-2">
					{#each improved as s}<li class="pill bg-leaf-tint text-leaf-deep">{s.name}</li>{/each}
				</ul>
			</div>
		{/if}

		{#if result.unlocks.length}
			<div class="card anim-rise mt-4 p-5 text-center" style="--d:1000ms">
				<p class="font-extrabold">Hadiah baru!</p>
				<ul class="mt-3 flex flex-wrap justify-center gap-5">
					{#each result.unlocks as u, i}<li class="flex flex-col items-center gap-1"><span class="anim-pop grid size-16 place-items-center rounded-2xl bg-sun-tint" style="--d:{1150 + i * 150}ms"><Figure name={u.icon} size={40} /></span><span class="text-sm font-bold">{u.name}</span></li>{/each}
				</ul>
			</div>
		{/if}

		<div class="anim-rise mt-6 flex flex-col gap-3 sm:flex-row" style="--d:1100ms">
			<a href="/app/play/{childId}" class="btn btn-primary flex-1 text-lg">Selesai</a>
			<a href="/app/play/{childId}/rewards" class="btn btn-quiet flex-1 text-lg">Lihat hadiah</a>
		</div>
	</section>
{:else if phase === 'mini'}
	<section class="mx-auto mt-10 max-w-md text-center" aria-live="polite">
		<div class="relative mx-auto w-48"><img src={wellDone} alt="" class="anim-pop w-48" /><Burst count={14} spread={110} /></div>
		<h1 class="anim-rise mt-4 text-4xl" style="--d:150ms">Tiga soal selesai!</h1>
		<p class="anim-rise mt-2 text-lg text-ink-soft" style="--d:250ms">Tinggal {asm.planned_count - answeredCount} lagi. Kamu pasti bisa.</p>
		<button class="btn btn-primary mt-8 min-h-16 px-12 text-xl" onclick={() => { phase = 'final'; next(); }}>Lanjut</button>
	</section>
{:else}
	<div class="flex items-center gap-3 pt-2">
		<a href="/app/play/{childId}" class="tile grid size-12 shrink-0 place-items-center text-ink-soft" aria-label="Berhenti dulu, bisa dilanjutkan nanti"><X size={20} weight="bold" aria-hidden="true" /></a>
		<ol class="flex flex-1 gap-1.5" aria-label="Soal {Math.min(answeredCount + 1, asm.planned_count)} dari {asm.planned_count}">
			{#each Array(asm.planned_count) as _, i}
				<li class="h-3 flex-1 rounded-full transition-colors duration-500 {i < answeredCount ? (asm.results[i] ? 'bg-leaf' : 'bg-sun') : i === answeredCount ? 'bg-primary' : 'bg-line'}"></li>
			{/each}
		</ol>
	</div>
	{#if asm.focus_skill}<p class="pill mt-3 bg-tangerine-tint text-tangerine-deep"><Flag size={14} weight="fill" aria-hidden="true" />Latihan: {asm.focus_skill}</p>{/if}

	{#if item}
		{#key item.id}
			<section class="card anim-rise-scale mt-4 p-5 sm:p-8" aria-label="Soal">
				<QuestionRenderer type={item.type} content={item.content} bind:response disabled={phase === 'final' || busy} {reveal} band={asm.ux_band} onsubmit={submit} />
			</section>
		{/key}

		{#if hint && phase !== 'final'}
			<p class="anim-rise mt-4 flex items-start gap-3 rounded-2xl bg-sun-tint p-4 text-lg font-semibold" role="status">
				<Lightbulb size={26} weight="fill" class="shrink-0 text-sun-deep" aria-hidden="true" />{hint}
			</p>
		{/if}
	{/if}

	<!-- Sticky action bar: stays above the home indicator and never covers content (§33). -->
	<div class="h-44" aria-hidden="true"></div>
	<div class="safe-bottom fixed inset-x-0 bottom-0 z-10 border-t border-line px-4 pt-3 transition-colors duration-300 {feedback ? (feedback.ok ? 'bg-leaf-tint' : 'bg-sun-tint') : 'bg-paper'}" aria-live="polite">
		<div class="mx-auto max-w-4xl">
			{#if feedback}
				{#key feedback}
				<div class="mb-3 flex items-start gap-3 {feedback.ok ? 'anim-rise' : phase === 'final' ? 'anim-rise' : 'anim-nudge'}">
					<span class="relative grid size-11 shrink-0 place-items-center rounded-full {feedback.ok ? 'anim-pop' : ''} {feedback.ok ? 'bg-leaf-deep text-white' : phase === 'final' ? 'bg-primary text-white' : 'bg-sun-deep text-ink'}" aria-hidden="true">
						{#if feedback.ok}<CheckCircle size={26} weight="fill" /><Burst />{:else if phase === 'final'}<Info size={26} weight="fill" />{:else}<ArrowCounterClockwise size={24} weight="bold" />{/if}
					</span>
					<div>
						<p class="text-xl font-extrabold">{feedback.text}</p>
						{#if phase === 'final' && explanation}<p class="mt-0.5 text-ink-soft">{explanation}</p>{/if}
					</div>
				</div>
				{/key}
			{/if}
			{#if error}<p class="field-error mb-2" role="alert">{error}</p>{/if}
			<div class="flex gap-3">
				{#if phase === 'final' || phase === 'finishing'}
					<button class="btn btn-primary min-h-16 flex-1 text-xl" onclick={next} disabled={phase === 'finishing'}>
						{phase === 'finishing' ? 'Menyimpan…' : done ? 'Lihat hasil' : 'Lanjut'}
					</button>
				{:else}
					{#if !hint}
						<button class="btn btn-quiet min-h-16 text-lg" onclick={askHint}><Lightbulb size={24} weight="fill" class="text-sun-deep" aria-hidden="true" /><span class="max-sm:sr-only">Petunjuk</span></button>
					{/if}
					<button class="btn btn-primary min-h-16 flex-1 text-xl" disabled={!response || busy || JSON.stringify(response) === lastWrong} onclick={submit}>
						{busy ? 'Memeriksa…' : phase === 'retry' ? 'Coba lagi' : 'Cek jawaban'}
					</button>
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	/* One orchestrated moment per event: stars land on the result, a pop on correct. */
	.stars span {
		display: inline-block;
		animation: land 0.45s cubic-bezier(0.3, 1.6, 0.5, 1) both;
		animation-delay: calc(var(--i) * 0.15s);
	}
	@keyframes land {
		from { transform: scale(0.2) rotate(-25deg); opacity: 0; }
		to { transform: none; opacity: 1; }
	}
</style>
