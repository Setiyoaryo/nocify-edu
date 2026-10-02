<script lang="ts">
	// Reusable inline review form for a pipeline candidate (question or material).
	// Reuses POST /admin/candidates/{kind}/{id}/decision, so server-side
	// ValidateDecision + checklist enforcement is unchanged.
	import { get, post, ApiError } from '$lib/api';
	import { dateTime } from '$lib/format';
	import { session } from '$lib/session.svelte';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import type { CandidateKind, Publication } from '$lib/candidates';

	let {
		kind,
		id,
		publication = null,
		title = 'Keputusan review',
		ondecided
	}: {
		kind: CandidateKind;
		id: number;
		publication?: Publication | null;
		title?: string;
		ondecided?: (pub: Publication) => void;
	} = $props();

	// Checklist definitions are static for the session; share one fetch across
	// every decision form on the page.
	let defsPromise: Promise<{ Key: string; Label: string }[]> | null = null;
	function loadDefs() {
		if (!defsPromise) {
			defsPromise = get('/admin/question-types')
				.then((r) => r.checklist as { Key: string; Label: string }[])
				.catch((e) => {
					defsPromise = null;
					throw e;
				});
		}
		return defsPromise;
	}

	let checklistDefs = $state<{ Key: string; Label: string }[]>([]);
	let checklistErr = $state('');
	let checklist = $state<Record<string, boolean>>({});
	let notes = $state('');
	let msg = $state('');
	let err = $state('');
	let busy = $state(false);

	$effect(() => {
		let alive = true;
		loadDefs()
			.then((d) => {
				if (alive) checklistDefs = d;
			})
			.catch(() => {
				if (alive) checklistErr = 'Gagal memuat daftar quality gate.';
			});
		return () => {
			alive = false;
		};
	});

	const isReviewer = $derived(session.user?.role === 'reviewer' || session.user?.role === 'admin');
	const pub = $derived(publication);
	const locked = $derived(pub?.status === 'published' || pub?.status === 'publishing');
	const recorded = $derived(
		pub
			? Object.entries(pub.checklist ?? {})
					.filter(([, checked]) => checked)
					.map(([key]) => ({ key, label: checklistDefs.find((c) => c.Key === key)?.Label ?? key }))
			: []
	);

	async function decide(decision: 'approved' | 'rejected' | 'changes_requested') {
		busy = true;
		msg = err = '';
		try {
			const r = await post(`/admin/candidates/${kind}/${id}/decision`, { decision, checklist, notes });
			notes = '';
			msg =
				decision === 'approved'
					? kind === 'question'
						? 'Disetujui. Soal akan diterbitkan saat publish dijalankan.'
						: 'Disetujui. Materi belum diterbitkan (menunggu model materi kanonik).'
					: decision === 'rejected'
						? 'Kandidat ditolak.'
						: 'Perbaikan diminta.';
			ondecided?.(r.publication as Publication);
		} catch (e) {
			err = e instanceof ApiError ? e.message : 'Gagal menyimpan keputusan.';
		} finally {
			busy = false;
		}
	}
</script>

<section class="rounded-2xl border border-line p-4" aria-label={title}>
	<h3 class="font-bold">{title}</h3>
	{#if pub}
		<div class="mt-2 flex flex-wrap items-center gap-2 text-sm">
			<StatusPill status={pub.status} />
			<span class="text-ink-soft">oleh {pub.reviewer ?? '—'}, {dateTime(pub.decided_at)}</span>
		</div>
		{#if pub.canonical_question_id}
			<p class="mt-2 text-sm">
				Soal kanonik: <a class="font-mono text-primary-deep hover:underline" href="/app/admin/questions/{pub.canonical_question_id}">{pub.canonical_question_id}</a>
			</p>
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
		<p class="mt-2 text-sm text-ink-soft">Belum ada keputusan.</p>
	{/if}

	{#if locked}
		<p class="mt-3 text-sm text-ink-soft">Kandidat sudah dipublikasikan; keputusan tidak dapat diubah lagi.</p>
	{:else if isReviewer}
		{#if checklistDefs.length}
			<fieldset class="mt-3">
				<legend class="text-sm font-bold">Quality gate (wajib untuk menyetujui)</legend>
				<div class="mt-2 grid gap-x-6 gap-y-1 sm:grid-cols-2">
					{#each checklistDefs as c}
						<label class="flex min-h-11 items-center gap-3 text-sm"><input type="checkbox" class="size-5 accent-primary" bind:checked={checklist[c.Key]} />{c.Label}</label>
					{/each}
				</div>
			</fieldset>
		{:else}
			<p class="mt-3 rounded-xl bg-rose-tint p-3 text-sm text-[#a8264f]" role="alert">
				{checklistErr || 'Daftar quality gate tidak tersedia.'} Persetujuan tidak dapat dilakukan sampai daftar ini dimuat.
			</p>
		{/if}
		<label class="mt-3 block"><span class="label">Catatan (opsional)</span><input class="field" bind:value={notes} /></label>
		<div class="mt-3 flex flex-wrap gap-2">
			<button class="btn btn-primary" disabled={busy || !checklistDefs.length} onclick={() => decide('approved')}>Setujui</button>
			<button class="btn btn-quiet" disabled={busy} onclick={() => decide('changes_requested')}>Minta perbaikan</button>
			<button class="btn btn-danger" disabled={busy} onclick={() => decide('rejected')}>Tolak</button>
		</div>
		{#if err}<p class="field-error mt-3" role="alert">{err}</p>{/if}
		{#if msg}<p class="mt-3 font-semibold text-leaf-deep" role="status">{msg}</p>{/if}
	{:else}
		<p class="mt-3 text-sm text-ink-soft">Hanya reviewer atau admin yang dapat mengambil keputusan.</p>
	{/if}
</section>
