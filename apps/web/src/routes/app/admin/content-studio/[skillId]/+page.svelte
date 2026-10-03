<script lang="ts">
	// Content Studio — one lesson (skill): material candidates with the questions
	// derived from each (lineage = same chunk), plus already-published canonical
	// questions. Inline review reuses POST /admin/candidates/{kind}/{id}/decision.
	import { get, ApiError } from '$lib/api';
	import { page } from '$app/state';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import CandidateDecision from '$lib/components/CandidateDecision.svelte';
	import QuestionRenderer from '$lib/questions/QuestionRenderer.svelte';
	import { bandFor, type CandidateDetail, type CandidateKind, type Publication } from '$lib/candidates';
	import {
		hasAudio,
		hasVisual,
		needsVisual,
		visualKindLabel,
		visualKindOf,
		visualSummaryOf,
		type CanonicalQuestion,
		type LessonDetail,
		type StudioMaterialGroup
	} from '$lib/contentStudio';

	const skillId = page.params.skillId;
	let detail = $state<LessonDetail | null>(null);
	let err = $state('');
	let onlyNeedsVisual = $state(false);

	const materialsView = $derived(
		(detail?.materials ?? [])
			.map((g) => ({ ...g, questions: g.questions.filter((q) => !onlyNeedsVisual || needsVisual(q)) }))
			.filter((g) => !onlyNeedsVisual || g.questions.length > 0)
	);
	const orphanView = $derived((detail?.orphan_questions ?? []).filter((q) => !onlyNeedsVisual || needsVisual(q)));

	get(`/admin/content-studio/lessons/${skillId}`)
		.then((r: LessonDetail) => (detail = r))
		.catch(
			(e) =>
				(err =
					e instanceof ApiError
						? e.status === 404
							? 'Pelajaran tidak ditemukan.'
							: e.message
						: 'Gagal memuat pelajaran.')
		);

	function setPublication(kind: CandidateKind, id: number, pub: Publication) {
		if (!detail) return;
		for (const g of detail.materials) {
			if (g.material.kind === kind && g.material.id === id) g.material.publication = pub;
			for (const q of g.questions) if (q.kind === kind && q.id === id) q.publication = pub;
		}
		for (const q of detail.orphan_questions) if (q.kind === kind && q.id === id) q.publication = pub;
	}

	const readabilityLabels: Record<string, string> = {
		age_band: 'Kelompok usia',
		reading_level: 'Tingkat baca',
		complexity: 'Kompleksitas',
		word_count: 'Jumlah kata'
	};
</script>

{#snippet visualPill(hasV: boolean, hasA: boolean, content: Record<string, unknown> | undefined)}
	{@const kind = visualKindOf(hasV, hasA, content)}
	{@const summary = visualSummaryOf(content)}
	<span
		class="pill {kind === 'image' || kind === 'icon'
			? 'bg-leaf-tint text-leaf-deep'
			: kind === 'audio'
				? 'bg-sun-tint text-ink'
				: 'bg-paper-deep text-ink-soft'}"
	>
		{visualKindLabel(kind)}{#if summary && (kind === 'image' || kind === 'icon')} · {summary}{/if}
	</span>
{/snippet}

{#snippet questionCard(q: CandidateDetail)}
	<div class="rounded-2xl border border-line p-4">
		<div class="flex flex-wrap items-center gap-2">
			<span class="pill bg-paper-deep text-ink-soft">{q.type || 'soal'}</span>
			{#if q.difficulty}<span class="text-xs text-ink-soft">Kesulitan {q.difficulty}/5</span>{/if}
			{@render visualPill(hasVisual(q), hasAudio(q), q.payload?.content)}
			<span class="ml-auto font-mono text-xs text-ink-soft">#{q.id}</span>
		</div>
		{#if q.payload}
			<div class="mt-3 rounded-xl bg-paper-deep/50 p-3 sm:p-4">
				<QuestionRenderer type={q.payload.type} content={q.payload.content as any} response={null} disabled band={bandFor(q.mapping.phase_code)} />
			</div>
			<details class="mt-2">
				<summary class="cursor-pointer text-sm font-bold text-primary-deep">Kunci jawaban & penjelasan</summary>
				<pre class="mt-2 overflow-x-auto rounded-xl bg-night p-3 text-xs text-white">{JSON.stringify(q.payload.answer, null, 2)}</pre>
				{#if q.payload.explanation}<p class="mt-2 text-sm text-ink-soft">{q.payload.explanation}</p>{/if}
			</details>
		{:else}
			<p class="mt-2 text-sm text-ink-soft">Pratinjau soal tidak tersedia.</p>
		{/if}
		{#if q.mapping.skill_name}
			<p class="mt-2 text-xs text-ink-soft">Skill: {q.mapping.skill_name} · Fase {q.mapping.phase_name || q.mapping.phase_code || '—'}</p>
		{/if}
		<div class="mt-3">
			<CandidateDecision
				kind={q.kind}
				id={q.id}
				publication={q.publication}
				title="Review soal"
				ondecided={(pub) => setPublication('question', q.id, pub)}
			/>
		</div>
	</div>
{/snippet}

{#snippet canonicalCard(p: CanonicalQuestion)}
	<div class="rounded-2xl border border-line p-4">
		<div class="flex flex-wrap items-center gap-2">
			<span class="pill bg-leaf-tint text-leaf-deep">Tayang</span>
			<span class="pill bg-paper-deep text-ink-soft">{p.type || 'soal'}</span>
			{#if p.difficulty}<span class="text-xs text-ink-soft">Kesulitan {p.difficulty}/5</span>{/if}
			{@render visualPill(p.has_visual, p.has_audio, p.content)}
			<span class="ml-auto font-mono text-xs text-ink-soft">{p.code}</span>
		</div>
		<div class="mt-3 rounded-xl bg-paper-deep/50 p-3 sm:p-4">
			<QuestionRenderer type={p.type} content={p.content as any} response={null} disabled />
		</div>
		<details class="mt-2">
			<summary class="cursor-pointer text-sm font-bold text-primary-deep">Kunci jawaban & penjelasan</summary>
			<pre class="mt-2 overflow-x-auto rounded-xl bg-night p-3 text-xs text-white">{JSON.stringify(p.answer, null, 2)}</pre>
			{#if p.explanation}<p class="mt-2 text-sm text-ink-soft">{p.explanation}</p>{/if}
		</details>
		<div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-ink-soft">
			{#if p.reviewer}<span>Diterbitkan oleh {p.reviewer}</span>{/if}
			<a class="font-bold text-primary-deep hover:underline" href="/app/admin/questions/{p.id}">Buka di bank soal</a>
		</div>
	</div>
{/snippet}

<svelte:head><title>{detail?.skill.name || 'Pelajaran'} | Content Studio</title></svelte:head>
<a href="/app/admin/content-studio" class="text-sm font-bold text-primary-deep hover:underline">Kembali ke Content Studio</a>

{#if err}
	<div class="card mt-4 p-6" role="alert">
		<p class="font-semibold text-[#a8264f]">{err}</p>
		<p class="mt-1 text-sm text-ink-soft">Kembali ke Content Studio untuk memilih pelajaran lain.</p>
	</div>
{:else if detail}
	<header class="mt-2">
		<p class="text-xs font-bold text-ink-soft uppercase">
			{detail.skill.grade_name} · {detail.skill.subject_name}{#if detail.skill.chapter_name} · {detail.skill.chapter_name}{/if}
		</p>
		<h1 class="mt-1 text-3xl sm:text-4xl">{detail.skill.name || detail.skill.code}</h1>
		<p class="mt-1 font-mono text-sm text-ink-soft">{detail.skill.code}</p>
		{#if detail.skill.description}<p class="mt-2 text-ink-soft">{detail.skill.description}</p>{/if}
	</header>

	<label class="mt-4 inline-flex min-h-11 items-center gap-3">
		<input type="checkbox" class="size-5 accent-primary" bind:checked={onlyNeedsVisual} />
		<span class="text-sm font-semibold">Hanya soal yang butuh visual</span>
	</label>

	{#if !detail.available}
		<div class="card mt-6 p-6">
			<p class="text-ink-soft">Pipeline konten belum dijalankan di lingkungan ini.</p>
			<p class="mt-1 text-sm text-ink-soft">Soal yang sudah tayang tetap ditampilkan di bawah.</p>
		</div>
	{/if}

	<!-- 1. Pipeline materials + derived questions -->
	<section class="mt-8" aria-labelledby="materi">
		<h2 id="materi" class="text-2xl">Materi pipeline</h2>
		<p class="mt-1 text-sm text-ink-soft">
			Materi bersifat review-only: yang tayang ke kelas adalah soal turunannya. Soal dikelompokkan berdasarkan chunk sumber yang sama.
		</p>
		{#if materialsView.length}
			<div class="mt-4 space-y-6">
				{#each materialsView as g (g.material.id)}
					{@const m = g.material}
					{@const readability = (m.readability ?? null) as Record<string, unknown> | null}
					<div class="card p-5 sm:p-6">
						<div class="flex flex-wrap items-center gap-2">
							<span class="pill bg-paper-deep text-ink-soft">Materi</span>
							<StatusPill status={m.publication?.status ?? m.status} />
							<span class="font-mono text-xs text-ink-soft">{m.source.code}</span>
							<span class="ml-auto text-xs text-ink-soft">chunk #{m.lineage.chunk_id}</span>
						</div>
						<h3 class="mt-2 text-xl">{m.title || m.prompt || `Materi #${m.id}`}</h3>
						<p class="text-sm text-ink-soft">
							{m.lineage.document_title || '—'}{#if m.lineage.heading} · {m.lineage.heading}{/if}
							{#if m.lineage.page_start} · Hal. {m.lineage.page_start}{#if m.lineage.page_end && m.lineage.page_end !== m.lineage.page_start}–{m.lineage.page_end}{/if}{/if}
						</p>
						{#if m.summary}<p class="mt-2">{m.summary}</p>{/if}
						{#if m.learning_objective}<p class="mt-2 text-sm"><b>Tujuan:</b> {m.learning_objective}</p>{/if}
						<div class="mt-2 flex flex-wrap gap-2 text-xs">
							{#if readability}
								{#each Object.entries(readabilityLabels) as [k, label]}
									{#if readability[k] !== undefined && readability[k] !== ''}
										<span class="pill bg-paper-deep text-ink-soft">{label}: {readability[k]}</span>
									{/if}
								{/each}
								{#if readability.visual_required}<span class="pill bg-sun-tint text-ink">Butuh visual</span>{/if}
								{#if readability.audio_required}<span class="pill bg-sun-tint text-ink">Butuh audio</span>{/if}
							{/if}
						</div>
						<details class="mt-3">
							<summary class="cursor-pointer text-sm font-bold text-primary-deep">Kutipan sumber</summary>
							<p class="mt-2 rounded-xl bg-paper-deep/50 p-3 text-sm whitespace-pre-wrap">{m.lineage.text}</p>
							{#if m.lineage.canonical_url}
								<a class="mt-1 inline-block break-all text-xs text-primary-deep hover:underline" href={m.lineage.canonical_url} target="_blank" rel="noreferrer">{m.lineage.canonical_url}</a>
							{/if}
						</details>
						<div class="mt-3">
							<CandidateDecision
								kind="material"
								id={m.id}
								publication={m.publication}
								title="Review materi"
								ondecided={(pub) => setPublication('material', m.id, pub)}
							/>
						</div>

						<h4 class="mt-6 text-lg">Soal turunan ({g.questions.length})</h4>
						{#if g.questions.length}
							<div class="mt-2 space-y-4">
								{#each g.questions as q (q.id)}{@render questionCard(q)}{/each}
							</div>
						{:else}
							<p class="mt-2 text-sm text-ink-soft">Belum ada soal turunan dari materi ini.</p>
						{/if}
					</div>
				{/each}
			</div>
		{:else}
			<p class="mt-4 text-ink-soft">
				{onlyNeedsVisual
					? 'Tidak ada soal yang butuh visual di materi pipeline.'
					: 'Tidak ada materi pipeline untuk pelajaran ini.'}
			</p>
		{/if}
	</section>

	<!-- 2. Questions with no material -->
	{#if orphanView.length}
		<section class="mt-10" aria-labelledby="orphan">
			<h2 id="orphan" class="text-2xl">Soal tanpa materi ({orphanView.length})</h2>
			<p class="mt-1 text-sm text-ink-soft">Soal ini tidak punya chunk sumber yang cocok, jadi tidak dikelompokkan di bawah materi mana pun.</p>
			<div class="mt-4 space-y-4">
				{#each orphanView as q (q.id)}{@render questionCard(q)}{/each}
			</div>
		</section>
	{/if}

	<!-- 3. Already-published canonical questions -->
	<section class="mt-10" aria-labelledby="tayang">
		<h2 id="tayang" class="text-2xl">Sudah tayang ({detail.published.length})</h2>
		<p class="mt-1 text-sm text-ink-soft">Soal kanonik yang sudah diterbitkan ke bank soal. Tidak ada keputusan review di sini.</p>
		{#if detail.published.length}
			<div class="mt-4 space-y-4">
				{#each detail.published as p (p.id)}{@render canonicalCard(p)}{/each}
			</div>
		{:else}
			<p class="mt-4 text-ink-soft">Belum ada soal yang tayang untuk pelajaran ini.</p>
		{/if}
	</section>
{:else}
	<div class="skeleton mt-6 h-96" aria-busy="true"></div>
{/if}
