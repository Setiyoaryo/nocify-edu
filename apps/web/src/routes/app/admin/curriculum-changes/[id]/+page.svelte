<script lang="ts">
	import { get, post, ApiError } from '$lib/api';
	import { page } from '$app/state';
	import { session } from '$lib/session.svelte';
	import { dateTime } from '$lib/format';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import {
		changeTypeLabel,
		eventStatusLabel,
		matchLabel,
		reviewStatusLabel,
		shortHash,
		type ChangeDetail,
		type ReviewStatus
	} from '$lib/curriculumChanges';

	const id = page.params.id;

	let data = $state<ChangeDetail | null>(null);
	let loadErr = $state('');
	let notes = $state('');
	let msg = $state('');
	let err = $state('');
	let busy = $state(false);

	async function load() {
		loadErr = '';
		try {
			const r = await get(`/admin/curriculum-changes/${id}`);
			data = r.change;
			notes = '';
		} catch (e) {
			loadErr = e instanceof ApiError && e.status === 404
				? 'Perubahan tidak ditemukan. Pipeline konten mungkin belum dijalankan di lingkungan ini.'
				: e instanceof ApiError ? e.message : 'Gagal memuat perubahan.';
		}
	}
	load();

	const isReviewer = $derived(session.user?.role === 'reviewer' || session.user?.role === 'admin');
	const decisions: [ReviewStatus, string][] = [
		['acknowledged', 'Terima'],
		['deferred', 'Tunda'],
		['dismissed', 'Abaikan']
	];

	async function decide(status: ReviewStatus) {
		busy = true;
		msg = err = '';
		try {
			const r = await post(`/admin/curriculum-changes/${id}/decision`, { status, notes });
			data = { ...data!, review: r.review, review_status: r.review.status, reviewer: r.review.reviewer, decided_at: r.review.decided_at };
			notes = '';
			msg = `Tinjauan disimpan: ${reviewStatusLabel(status)}.`;
		} catch (e) {
			err = e instanceof ApiError ? e.message : 'Gagal menyimpan tinjauan.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>{data ? data.document_code : 'Perubahan kurikulum'} | CMS</title></svelte:head>
<a href="/app/admin/curriculum-changes" class="text-sm font-bold text-primary-deep hover:underline">Kembali ke daftar perubahan</a>

{#if data}
	<div class="mt-2 flex flex-wrap items-center gap-3">
		<h1 class="font-mono text-2xl sm:text-3xl">{data.document_code}</h1>
		<StatusPill status={data.review_status} />
		<span class="pill bg-paper-deep text-ink-soft">{eventStatusLabel(data.event_status)}</span>
	</div>
	<p class="mt-1 break-all text-sm text-ink-soft">
		<a class="text-primary-deep hover:underline" href={data.url} target="_blank" rel="noreferrer">{data.url}</a>
	</p>

	<div class="mt-6 grid gap-6 lg:grid-cols-[1.6fr_1fr]">
		<div class="space-y-6">
			<section class="card p-5 sm:p-6" aria-labelledby="impacts">
				<h2 id="impacts" class="text-xl">Capaian Pembelajaran terdampak</h2>
				<p class="mt-1 text-sm text-ink-soft">{data.cp_added} ditambahkan · {data.cp_changed} diubah · {data.cp_removed} dihapus · {data.affected_questions} soal terdampak</p>
				<ul class="mt-4 space-y-4">
					{#each data.impacts as impact (impact.id)}
						<li class="rounded-2xl border border-line p-4">
							<div class="flex flex-wrap items-center gap-2">
								<span class="font-mono font-bold">{impact.cp_code}</span>
								<span class="pill {impact.change_type === 'added' ? 'bg-sky-tint text-[#1a6fa3]' : impact.change_type === 'removed' ? 'bg-rose-tint text-[#a8264f]' : 'bg-sun-tint text-ink'}">{changeTypeLabel(impact.change_type)}</span>
								<span class="text-xs text-ink-soft">{impact.affected_skills} skill · {impact.affected_questions} soal</span>
							</div>
							{#if impact.change_type !== 'added'}
								<div class="mt-3 text-sm">
									<p class="text-xs font-bold text-ink-soft">Sebelum</p>
									<p class="text-ink-soft line-through">{impact.old_statement || '—'}</p>
								</div>
							{/if}
							{#if impact.change_type !== 'removed'}
								<div class="mt-3 text-sm">
									<p class="text-xs font-bold text-ink-soft">Sesudah</p>
									<p>{impact.new_statement || '—'}</p>
								</div>
							{/if}
							{#if impact.questions.length}
								<details class="mt-3">
									<summary class="cursor-pointer text-sm font-bold text-primary-deep">Soal terdampak ({impact.questions.length})</summary>
									<ul class="mt-2 space-y-1 text-sm">
										{#each impact.questions as q (q.question_id)}
											<li class="flex flex-wrap items-center gap-2">
												<a class="font-mono text-primary-deep hover:underline" href="/app/admin/questions/{q.question_id}">{q.question_code || q.question_id}</a>
												<span class="text-ink-soft">{q.skill_name || `skill #${q.skill_id}`}</span>
												<span class="pill bg-paper-deep text-ink-soft">{matchLabel(q.match)}</span>
												{#if q.status}<span class="text-xs text-ink-soft">{q.status}</span>{/if}
											</li>
										{/each}
									</ul>
								</details>
							{/if}
						</li>
					{:else}
						<li class="rounded-2xl border border-line p-6 text-center text-ink-soft">Tidak ada CP yang berubah pada peristiwa ini.</li>
					{/each}
				</ul>
			</section>
		</div>

		<div class="space-y-6">
			<section class="card p-5 sm:p-6" aria-labelledby="det">
				<h2 id="det" class="text-xl">Deteksi</h2>
				<dl class="mt-3 space-y-2 text-sm">
					<div><dt class="text-ink-soft">Terdeteksi</dt><dd class="font-semibold">{dateTime(data.created_at)}</dd></div>
					<div><dt class="text-ink-soft">Metode</dt><dd>{data.detection}</dd></div>
					{#if data.document_type}<div><dt class="text-ink-soft">Jenis dokumen</dt><dd>{data.document_type}</dd></div>{/if}
					<div><dt class="text-ink-soft">Hash dasar</dt><dd class="font-mono text-xs" title={data.baseline_hash}>{shortHash(data.baseline_hash)}</dd></div>
					<div><dt class="text-ink-soft">Hash teramati</dt><dd class="font-mono text-xs" title={data.observed_hash}>{shortHash(data.observed_hash)}</dd></div>
				</dl>
				{#if data.error}<p class="mt-3 rounded-xl bg-rose-tint p-2 text-xs text-[#a8264f]">{data.error}</p>{/if}
				<p class="mt-3 text-xs text-ink-soft">M7 hanya mendeteksi dan mencatat tinjauan. Kurikulum kanonik tidak diubah oleh pipeline.</p>
			</section>

			<section class="card p-5 sm:p-6" aria-labelledby="rev">
				<h2 id="rev" class="text-xl">Tinjauan</h2>
				{#if data.review}
					<div class="mt-3 flex flex-wrap items-center gap-2 text-sm">
						<StatusPill status={data.review.status} />
						<span class="text-ink-soft">oleh {data.review.reviewer ?? '—'}{#if data.review.decided_at}, {dateTime(data.review.decided_at)}{/if}</span>
					</div>
					{#if data.review.notes}<p class="mt-2 text-sm text-ink-soft">Catatan: {data.review.notes}</p>{/if}
				{:else}
					<p class="mt-3 text-sm text-ink-soft">Belum ada tinjauan.</p>
				{/if}

				{#if isReviewer}
					<label class="mt-4 block"><span class="label">Catatan (opsional)</span><input class="field" bind:value={notes} /></label>
					<div class="mt-4 flex flex-wrap gap-2">
						{#each decisions as [value, label]}
							<button class="btn {value === 'acknowledged' ? 'btn-primary' : value === 'dismissed' ? 'btn-danger' : 'btn-quiet'}" disabled={busy} onclick={() => decide(value)}>{label}</button>
						{/each}
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
		<p class="mt-1 text-sm text-ink-soft">Kembali ke daftar perubahan untuk memilih peristiwa lain.</p>
	</div>
{:else}
	<div class="skeleton mt-6 h-96" aria-busy="true"></div>
{/if}
