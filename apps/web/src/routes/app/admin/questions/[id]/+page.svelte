<script lang="ts">
	import { get, post, put, ApiError } from '$lib/api';
	import { page } from '$app/state';
	import { session } from '$lib/session.svelte';
	import { dateTime } from '$lib/format';
	import QuestionEditor, { type Draft } from '../QuestionEditor.svelte';
	import type { Response } from '$lib/types';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	import XCircle from 'phosphor-svelte/lib/XCircle';

	const id = page.params.id;
	let data = $state<any>(null);
	let value = $state<Draft | null>(null);
	let fields = $state<Record<string, string>>({});
	let checklistDefs = $state<{ Key: string; Label: string }[]>([]);
	let checklist = $state<Record<string, boolean>>({});
	let notes = $state('');
	let msg = $state('');
	let err = $state('');
	let testResult = $state<boolean | null>(null);

	function fill(r: any) {
		data = r;
		const v = r.versions[0];
		value = {
			code: r.question.code, skill_id: r.question.skill_id, type: r.question.type, difficulty: r.question.difficulty,
			learning_objective: r.question.learning_objective, source: r.question.source,
			content: JSON.stringify(v.content, null, 2), answer: JSON.stringify(v.answer, null, 2), explanation: v.explanation, hint: v.hint
		};
		checklist = {};
		notes = '';
	}
	get(`/admin/questions/${id}`).then(fill);
	get('/admin/question-types').then((r) => (checklistDefs = r.checklist));

	const isReviewer = $derived(session.user?.role === 'reviewer' || session.user?.role === 'admin');
	const q = $derived(data?.question);
	const transitions: Record<string, { to: string; label: string; reviewer?: boolean; quiet?: boolean }[]> = {
		draft: [{ to: 'review', label: 'Kirim ke review' }],
		review: [{ to: 'approved', label: 'Setujui', reviewer: true }, { to: 'draft', label: 'Minta perbaikan', reviewer: true, quiet: true }],
		approved: [{ to: 'published', label: 'Tayangkan', reviewer: true }, { to: 'draft', label: 'Kembalikan ke draft', reviewer: true, quiet: true }],
		published: [{ to: 'retired', label: 'Arsipkan (berhenti tayang)', reviewer: true, quiet: true }],
		retired: [{ to: 'draft', label: 'Aktifkan kembali sebagai draft' }]
	};
	const statusLabel: Record<string, string> = { draft: 'Draft', review: 'Menunggu review', approved: 'Disetujui', published: 'Tayang', retired: 'Diarsipkan' };

	async function save() {
		if (!value) return;
		fields = {};
		msg = err = '';
		try {
			fill(await put(`/admin/questions/${id}`, { ...value, content: JSON.parse(value.content), answer: JSON.parse(value.answer) }));
			msg = 'Versi baru tersimpan sebagai draft.';
		} catch (e) {
			fields = e instanceof ApiError ? (Object.keys(e.fields).length ? e.fields : { content: e.message }) : { content: String(e) };
		}
	}
	async function transition(to: string) {
		msg = err = '';
		try {
			fill(await post(`/admin/questions/${id}/transition`, { to, checklist, notes }));
			msg = `Status: ${statusLabel[to]}.`;
		} catch (e) {
			err = e instanceof ApiError ? e.message : 'Gagal.';
		}
	}
	async function test(response: Response | null) {
		if (!response) return;
		testResult = (await post(`/admin/questions/${id}/preview`, { response })).correct;
	}
</script>

<svelte:head><title>{q?.code ?? 'Soal'} | CMS</title></svelte:head>
<a href="/app/admin/questions" class="text-sm font-bold text-primary-deep hover:underline">Kembali ke bank soal</a>

{#if q && value}
	<div class="mt-2 flex flex-wrap items-center gap-3">
		<h1 class="text-3xl">{q.code}</h1>
		<StatusPill status={q.status} />
		{#if q.published_version_id}
			<span class="text-sm text-ink-soft">Tayang: versi {data.versions.find((v: any) => v.id === q.published_version_id)?.version}</span>
		{/if}
	</div>

	<section class="card mt-6 p-5 sm:p-6" aria-labelledby="wf">
		<h2 id="wf" class="text-xl">Alur review</h2>
		<p class="mt-1 text-sm text-ink-soft">Tahapannya: draft, review, disetujui, tayang, lalu diarsipkan. Penulis versi tidak dapat menyetujui versinya sendiri.</p>
		{#if q.status === 'review' && isReviewer}
			<fieldset class="mt-4">
				<legend class="font-bold">Quality gate (wajib dicentang semua untuk menyetujui)</legend>
				<div class="mt-2 grid gap-x-6 gap-y-1 sm:grid-cols-2">
					{#each checklistDefs as c}
						<label class="flex min-h-11 items-center gap-3"><input type="checkbox" class="size-5 accent-primary" bind:checked={checklist[c.Key]} />{c.Label}</label>
					{/each}
				</div>
			</fieldset>
		{/if}
		{#if (transitions[q.status] ?? []).length}
			<label class="mt-4 block"><span class="label">Catatan (opsional)</span><input class="field" bind:value={notes} /></label>
		{/if}
		<div class="mt-4 flex flex-wrap gap-2">
			{#each transitions[q.status] ?? [] as t}
				{#if !t.reviewer || isReviewer}
					<button class="btn {t.quiet ? 'btn-quiet' : 'btn-primary'}" onclick={() => transition(t.to)}>{t.label}</button>
				{/if}
			{/each}
		</div>
		{#if err}<p class="field-error mt-3" role="alert">{err}</p>{/if}
		{#if msg}<p class="mt-3 font-semibold text-leaf-deep" role="status">{msg}</p>{/if}
	</section>

	<div class="mt-8">
		<QuestionEditor bind:value {fields}>
			{#snippet after(resp)}
				<div class="mt-3 flex items-center gap-3">
					<button class="btn btn-quiet !min-h-11" disabled={!resp} onclick={() => test(resp)}>Uji jawaban di server</button>
					{#if testResult !== null}<span class="flex items-center gap-1.5 font-bold {testResult ? 'text-leaf-deep' : 'text-tangerine-deep'}">{#if testResult}<CheckCircle size={18} weight="fill" aria-hidden="true" />Dinilai benar{:else}<XCircle size={18} weight="fill" aria-hidden="true" />Dinilai salah{/if}</span>{/if}
				</div>
				<p class="mt-1 text-xs text-ink-soft">Menguji versi terbaru yang tersimpan.</p>
			{/snippet}
		</QuestionEditor>
	</div>
	<div class="mt-8 flex items-center gap-3">
		<button class="btn btn-primary" onclick={save}>Simpan sebagai versi baru</button>
		<span class="text-sm text-ink-soft">Mengedit mengembalikan status ke Draft. Versi yang sedang tayang tetap tampil ke anak sampai versi baru ditayangkan.</span>
	</div>

	<section class="card mt-8 grid gap-8 p-5 sm:p-6 md:grid-cols-2">
		<div>
			<h2 class="text-xl">Riwayat review</h2>
			<ul class="mt-3 space-y-2 text-sm">
				{#each data.reviews as r}
					<li class="border-l-3 border-line pl-3"><b>{r.reviewer}</b>: dari {statusLabel[r.from]} ke {statusLabel[r.to]} (v{r.version}), {dateTime(r.created_at)}{#if r.notes}<br /><span class="text-ink-soft">{r.notes}</span>{/if}</li>
				{:else}<li class="text-ink-soft">Belum ada.</li>{/each}
			</ul>
		</div>
		<div>
			<h2 class="text-xl">Versi</h2>
			<ul class="mt-3 space-y-1 text-sm">
				{#each data.versions as v}<li>v{v.version} oleh {v.created_by ?? 'seeder'}, {dateTime(v.created_at)}</li>{/each}
			</ul>
		</div>
	</section>
{/if}
