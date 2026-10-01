<script lang="ts">
	import { get, post, ApiError } from '$lib/api';
	import { page } from '$app/state';
	import { session } from '$lib/session.svelte';
	import { dateTime, date } from '$lib/format';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import QuestionRenderer from '$lib/questions/QuestionRenderer.svelte';
	import { bandFor, kindLabel, confidenceLabel, type CandidateDetail, type QaIssue } from '$lib/candidates';
	import WarningCircle from 'phosphor-svelte/lib/WarningCircle';
	import Info from 'phosphor-svelte/lib/Info';

	const kind = page.params.kind;
	const id = page.params.id;

	let data = $state<CandidateDetail | null>(null);
	let loadErr = $state('');
	let checklistDefs = $state<{ Key: string; Label: string }[]>([]);
	let checklistErr = $state('');
	let checklist = $state<Record<string, boolean>>({});
	let notes = $state('');
	let msg = $state('');
	let err = $state('');
	let busy = $state(false);

	async function load() {
		loadErr = '';
		try {
			const r = await get(`/admin/candidates/${kind}/${id}`);
			data = r.candidate;
			checklist = {};
			notes = '';
		} catch (e) {
			// 404 also means "pipeline not run" in this environment (see ListCandidates).
			loadErr = e instanceof ApiError && e.status === 404
				? 'Kandidat tidak ditemukan. Pipeline konten mungkin belum dijalankan di lingkungan ini.'
				: e instanceof ApiError ? e.message : 'Gagal memuat kandidat.';
		}
	}
	load();
	get('/admin/question-types')
		.then((r) => (checklistDefs = r.checklist))
		.catch(() => (checklistErr = 'Gagal memuat daftar quality gate.'));

	const isReviewer = $derived(session.user?.role === 'reviewer' || session.user?.role === 'admin');
	const pub = $derived(data?.publication);
	const locked = $derived(pub?.status === 'published' || pub?.status === 'publishing');
	const issues = $derived((data?.qa_issues ?? []) as QaIssue[]);
	const blocking = $derived(issues.filter((i) => i.blocking));
	const advisory = $derived(issues.filter((i) => !i.blocking));
	const readability = $derived((data?.readability ?? null) as Record<string, unknown> | null);
	// Quality-gate items recorded on the decision, labelled via the checklist definitions.
	const recorded = $derived(
		pub
			? Object.entries(pub.checklist ?? {})
					.filter(([, checked]) => checked)
					.map(([key]) => ({ key, label: checklistDefs.find((c) => c.Key === key)?.Label ?? key }))
			: []
	);
	const readabilityLabels: Record<string, string> = {
		age_band: 'Kelompok usia', reading_level: 'Tingkat baca', complexity: 'Kompleksitas',
		word_count: 'Jumlah kata', sentence_count: 'Jumlah kalimat', avg_sentence_length: 'Rata-rata panjang kalimat'
	};

	async function decide(decision: 'approved' | 'rejected' | 'changes_requested') {
		busy = true;
		msg = err = '';
		try {
			const r = await post(`/admin/candidates/${kind}/${id}/decision`, { decision, checklist, notes });
			data = { ...data!, publication: r.publication };
			notes = '';
			msg = decision === 'approved'
				? (kind === 'question' ? 'Disetujui. Soal akan diterbitkan saat publish dijalankan.' : 'Disetujui. Materi belum diterbitkan (menunggu model materi kanonik).')
				: decision === 'rejected' ? 'Kandidat ditolak.' : 'Perbaikan diminta.';
		} catch (e) {
			err = e instanceof ApiError ? e.message : 'Gagal menyimpan keputusan.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>{data ? data.title || data.prompt : 'Kandidat'} | CMS</title></svelte:head>
<a href="/app/admin/review" class="text-sm font-bold text-primary-deep hover:underline">Kembali ke antrian review</a>

{#if data}
	<div class="mt-2 flex flex-wrap items-center gap-3">
		<h1 class="text-3xl">{kindLabel(data.kind)} #{data.id}</h1>
		<StatusPill status={data.status} />
		{#if data.ai_generated}
			<span class="pill bg-sun-tint text-ink" title="Dibuat oleh model AI; wajib review manusia (§24)">Dibuat AI — wajib review</span>
		{/if}
		<span class="text-sm text-ink-soft">Keyakinan pemetaan {confidenceLabel(data.confidence)}</span>
	</div>
	<p class="mt-1 text-sm text-ink-soft">
		Status kandidat di atas berasal dari pipeline. Keputusan review tercatat terpisah di bawah.
	</p>

	<div class="mt-6 grid gap-6 lg:grid-cols-[1.6fr_1fr]">
		<div class="space-y-6">
			<!-- Candidate -->
			<section class="card p-5 sm:p-6" aria-labelledby="cand">
				<h2 id="cand" class="text-xl">{data.kind === 'question' ? 'Pratinjau soal' : 'Materi'}</h2>
				{#if data.kind === 'question' && data.payload}
					<div class="mt-4 rounded-2xl bg-paper-deep/50 p-4 sm:p-5">
						<QuestionRenderer type={data.payload.type} content={data.payload.content as any} response={null} disabled band={bandFor(data.mapping.phase_code)} />
					</div>
					<details class="mt-4">
						<summary class="cursor-pointer text-sm font-bold text-primary-deep">Lihat kunci jawaban & penjelasan</summary>
						<pre class="mt-2 overflow-x-auto rounded-xl bg-night p-3 text-xs text-white">{JSON.stringify(data.payload.answer, null, 2)}</pre>
						<p class="mt-2 text-sm text-ink-soft">{data.payload.explanation}</p>
					</details>
				{:else}
					<p class="mt-3 text-lg font-bold">{data.title}</p>
					<p class="mt-1 text-ink-soft">{data.summary}</p>
					{#if data.learning_objective}<p class="mt-3 text-sm"><span class="font-bold">Tujuan:</span> {data.learning_objective}</p>{/if}
					<div class="mt-3 flex flex-wrap gap-x-6 gap-y-1 text-sm text-ink-soft">
						{#if data.difficulty}<span>Kesulitan {data.difficulty}/5</span>{/if}
						{#if data.language}<span>Bahasa {data.language}</span>{/if}
						{#if readability}
							{#each Object.entries(readabilityLabels) as [k, label]}
								{#if readability[k] !== undefined && readability[k] !== ''}<span>{label}: {readability[k]}</span>{/if}
							{/each}
							{#if readability.visual_required}<span>Butuh visual</span>{/if}
							{#if readability.audio_required}<span>Butuh audio</span>{/if}
						{/if}
					</div>
				{/if}
			</section>

			<!-- Source -->
			<section class="card p-5 sm:p-6" aria-labelledby="src">
				<h2 id="src" class="text-xl">Sumber</h2>
				<dl class="mt-3 grid gap-x-6 gap-y-2 text-sm sm:grid-cols-2">
					<div><dt class="text-ink-soft">Sumber</dt><dd class="font-semibold">{data.source.name} <span class="font-mono text-xs">({data.source.code})</span></dd></div>
					<div><dt class="text-ink-soft">Dokumen</dt><dd class="font-semibold">{data.lineage.document_title || '—'}</dd></div>
					<div class="sm:col-span-2"><dt class="text-ink-soft">URL</dt><dd><a class="break-all text-primary-deep hover:underline" href={data.lineage.canonical_url} target="_blank" rel="noreferrer">{data.lineage.canonical_url}</a></dd></div>
					<div><dt class="text-ink-soft">Diambil</dt><dd>{data.lineage.retrieved_at ? date(data.lineage.retrieved_at) : '—'}</dd></div>
					<div><dt class="text-ink-soft">Lisensi</dt><dd>{data.lineage.license || '—'}</dd></div>
				</dl>
				{#if data.lineage.heading}<p class="mt-4 font-bold">{data.lineage.heading}</p>{/if}
				{#if data.lineage.page_start}<p class="text-xs text-ink-soft">Halaman {data.lineage.page_start}{#if data.lineage.page_end && data.lineage.page_end !== data.lineage.page_start}–{data.lineage.page_end}{/if}</p>{/if}
				<p class="mt-2 rounded-xl bg-paper-deep/50 p-3 text-sm whitespace-pre-wrap">{data.lineage.text}</p>
			</section>
		</div>

		<div class="space-y-6">
			<!-- Curriculum mapping -->
			<section class="card p-5 sm:p-6" aria-labelledby="map">
				<h2 id="map" class="text-xl">Pemetaan kurikulum</h2>
				<dl class="mt-3 space-y-2 text-sm">
					<div><dt class="text-ink-soft">Skill</dt><dd class="font-semibold">{data.mapping.skill_name || '—'} {#if data.mapping.skill_code}<span class="font-mono text-xs">({data.mapping.skill_code})</span>{/if}</dd></div>
					<div><dt class="text-ink-soft">Capaian Pembelajaran</dt><dd class="font-semibold">{data.mapping.cp_code || '—'}</dd>{#if data.mapping.cp_statement}<p class="mt-1 text-ink-soft">{data.mapping.cp_statement}</p>{/if}</div>
					<div class="flex gap-6">
						<div><dt class="text-ink-soft">Fase</dt><dd>{data.mapping.phase_name || data.mapping.phase_code || '—'}</dd></div>
						<div><dt class="text-ink-soft">Kelas</dt><dd>{data.mapping.grade_name || data.mapping.grade_code || '—'}</dd></div>
					</div>
					{#if data.mapping.subject_name}<div><dt class="text-ink-soft">Mapel</dt><dd>{data.mapping.subject_name}</dd></div>{/if}
				</dl>
			</section>

			<!-- QA -->
			<section class="card p-5 sm:p-6" aria-labelledby="qa">
				<h2 id="qa" class="text-xl">Pemeriksaan otomatis</h2>
				{#if blocking.length}
					<ul class="mt-3 space-y-2 text-sm">
						{#each blocking as issue}
							<li class="flex items-start gap-2 text-[#a8264f]"><WarningCircle size={16} weight="fill" class="mt-0.5 shrink-0" aria-hidden="true" /><span><b>{issue.code}</b> — {issue.message}</span></li>
						{/each}
					</ul>
				{:else}
					<p class="mt-3 flex items-center gap-2 text-sm font-semibold text-leaf-deep"><Info size={16} weight="fill" aria-hidden="true" />Tidak ada masalah yang menghalangi.</p>
				{/if}
				{#if advisory.length}
					<ul class="mt-3 space-y-1 text-sm text-ink-soft">
						{#each advisory as issue}<li>{issue.code} — {issue.message}</li>{/each}
					</ul>
				{/if}
			</section>

			<!-- Publication + decision -->
			<section class="card p-5 sm:p-6" aria-labelledby="dec">
				<h2 id="dec" class="text-xl">Keputusan review</h2>
				{#if pub}
					<div class="mt-3 flex flex-wrap items-center gap-2 text-sm">
						<StatusPill status={pub.status} />
						<span class="text-ink-soft">oleh {pub.reviewer ?? '—'}, {dateTime(pub.decided_at)}</span>
					</div>
					{#if pub.canonical_question_id}
						<p class="mt-2 text-sm">Soal kanonik: <a class="font-mono text-primary-deep hover:underline" href="/app/admin/questions/{pub.canonical_question_id}">{pub.canonical_question_id}</a></p>
					{/if}
					{#if pub.attempts > 0 && pub.status !== 'published'}<p class="mt-1 text-xs text-ink-soft">Percobaan publish: {pub.attempts}</p>{/if}
					{#if pub.last_error}<p class="mt-2 rounded-xl bg-rose-tint p-2 text-xs text-[#a8264f]">{pub.last_error}</p>{/if}
					{#if pub.notes}<p class="mt-2 text-sm text-ink-soft">Catatan: {pub.notes}</p>{/if}
					{#if recorded.length}
						<details class="mt-3">
							<summary class="cursor-pointer text-sm font-bold text-primary-deep">Quality gate yang dicatat ({recorded.length})</summary>
							<ul class="mt-2 grid gap-x-6 gap-y-1 text-sm text-ink-soft sm:grid-cols-2">
								{#each recorded as r}<li>{r.label}</li>{/each}
							</ul>
						</details>
					{/if}
				{:else}
					<p class="mt-3 text-sm text-ink-soft">Belum ada keputusan.</p>
				{/if}

				{#if locked}
					<p class="mt-4 text-sm text-ink-soft">Kandidat sudah dipublikasikan; keputusan tidak dapat diubah lagi.</p>
				{:else if isReviewer}
					{#if checklistDefs.length}
						<fieldset class="mt-4">
							<legend class="font-bold">Quality gate (wajib untuk menyetujui)</legend>
							<div class="mt-2 grid gap-x-6 gap-y-1 sm:grid-cols-2">
								{#each checklistDefs as c}
									<label class="flex min-h-11 items-center gap-3"><input type="checkbox" class="size-5 accent-primary" bind:checked={checklist[c.Key]} />{c.Label}</label>
								{/each}
							</div>
						</fieldset>
					{:else}
						<p class="mt-4 rounded-xl bg-rose-tint p-3 text-sm text-[#a8264f]" role="alert">
							{checklistErr || 'Daftar quality gate tidak tersedia.'} Persetujuan tidak dapat dilakukan sampai daftar ini dimuat.
						</p>
					{/if}
					<label class="mt-4 block"><span class="label">Catatan (opsional)</span><input class="field" bind:value={notes} /></label>
					<div class="mt-4 flex flex-wrap gap-2">
						<button class="btn btn-primary" disabled={busy || !checklistDefs.length} onclick={() => decide('approved')}>Setujui</button>
						<button class="btn btn-quiet" disabled={busy} onclick={() => decide('changes_requested')}>Minta perbaikan</button>
						<button class="btn btn-danger" disabled={busy} onclick={() => decide('rejected')}>Tolak</button>
					</div>
					{#if err}<p class="field-error mt-3" role="alert">{err}</p>{/if}
					{#if msg}<p class="mt-3 font-semibold text-leaf-deep" role="status">{msg}</p>{/if}
				{:else}
					<p class="mt-4 text-sm text-ink-soft">Hanya reviewer atau admin yang dapat mengambil keputusan.</p>
				{/if}
			</section>
		</div>
	</div>
{:else if loadErr}
	<div class="card mt-6 p-6" role="alert">
		<p class="font-semibold text-[#a8264f]">{loadErr}</p>
		<p class="mt-1 text-sm text-ink-soft">Kembali ke antrian review untuk memilih kandidat lain.</p>
	</div>
{:else}
	<div class="skeleton mt-6 h-96" aria-busy="true"></div>
{/if}
