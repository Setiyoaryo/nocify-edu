<script lang="ts">
	import { get, post, ApiError } from '$lib/api';
	import { page } from '$app/state';
	import { dateTime } from '$lib/format';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import PipelineStepTracker from '$lib/components/PipelineStepTracker.svelte';
	import PipelineMetricsCard from '$lib/components/PipelineMetricsCard.svelte';
	import {
		isActive,
		jobKindLabel,
		stageLabel,
		stageDescription,
		extractJobMetrics,
		type PipelineJob
	} from '$lib/pipeline';
	import { onDestroy } from 'svelte';
	import ArrowLeft from 'phosphor-svelte/lib/ArrowLeft';
	import ArrowClockwise from 'phosphor-svelte/lib/ArrowClockwise';
	import Question from 'phosphor-svelte/lib/Question';

	const id = page.params.id;
	let job = $state<PipelineJob | null>(null);
	let loadErr = $state('');
	let actionErr = $state('');
	let busy = $state(false);
	let timer: ReturnType<typeof setTimeout>;

	async function load() {
		clearTimeout(timer);
		try {
			job = (await get(`/admin/pipeline/jobs/${id}`)).job;
			loadErr = '';
		} catch (e) {
			loadErr = e instanceof ApiError && e.status === 404
				? 'Job tidak ditemukan.'
				: e instanceof ApiError ? e.message : 'Gagal memuat job.';
		}
		if (job && isActive(job)) timer = setTimeout(load, 3000);
	}
	load();
	onDestroy(() => clearTimeout(timer));

	async function jobAction(action: 'cancel' | 'retry') {
		busy = true;
		actionErr = '';
		try {
			await post(`/admin/pipeline/jobs/${id}/${action}`);
			await load();
		} catch (e) {
			actionErr = e instanceof ApiError ? e.message : 'Aksi gagal.';
		} finally {
			busy = false;
		}
	}

	const stages = $derived(Object.entries(job?.progress?.stages ?? {}));
	const jobMetrics = $derived(job ? extractJobMetrics(job) : null);
</script>

<svelte:head><title>Job #{id} | CMS EduPlay</title></svelte:head>

<div class="mb-4">
	<a href="/app/admin/pipeline" class="inline-flex items-center gap-1.5 text-sm font-bold text-primary-deep hover:underline">
		<ArrowLeft size={16} weight="bold" aria-hidden="true" />
		Kembali ke pipeline
	</a>
</div>

{#if job}
	<div class="flex flex-wrap items-center justify-between gap-4">
		<div class="flex flex-wrap items-center gap-3">
			<h1 class="text-3xl font-black">Job #{job.id}</h1>
			<StatusPill status={job.status} />
			<span class="text-sm font-semibold text-ink-soft">{jobKindLabel[job.kind] ?? job.kind}</span>
		</div>

		<div class="flex flex-wrap items-center gap-2">
			{#if job.status === 'queued' || job.status === 'running'}
				<button class="btn btn-sm text-error" disabled={busy} onclick={() => jobAction('cancel')}>
					Batalkan job
				</button>
			{:else if job.status === 'failed' || job.status === 'cancelled'}
				{#if job.attempts < job.max_attempts}
					<button class="btn btn-quiet btn-sm" disabled={busy} onclick={() => jobAction('retry')}>
						<ArrowClockwise size={16} weight="bold" aria-hidden="true" />
						Ulangi job
					</button>
				{/if}
			{/if}
			<a href="/app/admin/review?kind=question" class="btn btn-primary btn-sm">
				<Question size={16} weight="bold" aria-hidden="true" />
				Antrian review
			</a>
			<button class="btn btn-quiet btn-sm" disabled={busy} onclick={load} title="Muat ulang status">
				<ArrowClockwise size={16} weight="bold" aria-hidden="true" />
				Segarkan
			</button>
		</div>
	</div>

	{#if actionErr}<p class="field-error mt-3" role="alert">{actionErr}</p>{/if}

	<!-- 1. Visual Progress Bar & Step Tracker -->
	<div class="mt-6">
		<PipelineStepTracker {job} />
	</div>

	<!-- 2. Metrics Summary for this Job -->
	<div class="mt-6">
		<PipelineMetricsCard
			jobMetrics={jobMetrics ?? undefined}
			title="Hasil Konkrit Eksekusi Job #{job.id}"
		/>
	</div>

	<!-- Meta & Stages Breakdown -->
	<div class="mt-6 grid gap-6 lg:grid-cols-2">
		<!-- Metadata Card -->
		<section class="card p-5 sm:p-6" aria-labelledby="meta">
			<h2 id="meta" class="text-xl font-bold">Informasi & Metadata</h2>
			<dl class="mt-4 space-y-2.5 text-sm">
				<div class="flex justify-between border-b border-line pb-2">
					<dt class="text-ink-soft">Jenis Job</dt>
					<dd class="font-bold font-mono">{job.kind}</dd>
				</div>
				<div class="flex justify-between border-b border-line pb-2">
					<dt class="text-ink-soft">Percobaan</dt>
					<dd class="font-bold">{job.attempts} dari {job.max_attempts}</dd>
				</div>
				<div class="flex justify-between border-b border-line pb-2">
					<dt class="text-ink-soft">Waktu Dibuat</dt>
					<dd class="font-mono">{dateTime(job.created_at)}</dd>
				</div>
				<div class="flex justify-between border-b border-line pb-2">
					<dt class="text-ink-soft">Waktu Mulai</dt>
					<dd class="font-mono">{job.started_at ? dateTime(job.started_at) : '—'}</dd>
				</div>
				<div class="flex justify-between border-b border-line pb-2">
					<dt class="text-ink-soft">Waktu Selesai</dt>
					<dd class="font-mono">{job.finished_at ? dateTime(job.finished_at) : '—'}</dd>
				</div>
			</dl>

			{#if job.error}
				<div class="mt-4 rounded-xl bg-rose-tint p-4 text-sm text-[#a8264f]" role="alert">
					<p class="font-bold">Pesan Kesalahan:</p>
					<p class="mt-1 font-mono text-xs break-all">{job.error}</p>
				</div>
			{/if}

			<details class="mt-4">
				<summary class="cursor-pointer text-sm font-bold text-primary-deep hover:underline">
					Lihat Parameter Eksekusi (JSON)
				</summary>
				<pre class="mt-2 overflow-x-auto rounded-xl bg-night p-3 text-xs text-white">{JSON.stringify(job.params, null, 2)}</pre>
			</details>
		</section>

		<!-- Tahapan Breakdown Card -->
		<section class="card p-5 sm:p-6" aria-labelledby="stages">
			<h2 id="stages" class="text-xl font-bold">Rincian Tahapan yang Selesai</h2>
			{#if stages.length}
				<ul class="mt-4 space-y-3">
					{#each stages as [name, result]}
						<li class="rounded-xl border border-line bg-paper-deep/40 p-4 transition-all">
							<div class="flex items-center justify-between">
								<span class="font-bold text-ink">{stageLabel[name] ?? name}</span>
								<span class="pill bg-leaf-tint text-leaf-deep text-xs font-bold">✓ Selesai</span>
							</div>
							{#if stageDescription[name]}
								<p class="mt-0.5 text-xs text-ink-soft">{stageDescription[name]}</p>
							{/if}

							<details class="mt-2">
								<summary class="cursor-pointer text-xs font-bold text-primary-deep hover:underline">
									Rincian Output ({Array.isArray(result) ? `${result.length} item` : 'Objek'})
								</summary>
								<pre class="mt-2 overflow-x-auto rounded-xl bg-paper p-3 text-xs text-ink font-mono">{JSON.stringify(result, null, 2)}</pre>
							</details>
						</li>
					{/each}
				</ul>
			{:else}
				<div class="mt-4 rounded-xl border border-dashed border-line p-6 text-center text-sm text-ink-soft">
					{#if job.status === 'running'}
						Tahap pertama sedang diproses oleh worker...
					{:else if job.status === 'queued'}
						Menunggu giliran eksekusi worker.
					{:else}
						Belum ada tahap yang selesai.
					{/if}
				</div>
			{/if}
		</section>
	</div>
{:else if loadErr}
	<div class="card mt-6 p-6" role="alert">
		<p class="font-semibold text-[#a8264f]">{loadErr}</p>
	</div>
{:else}
	<div class="skeleton mt-6 h-96" aria-busy="true"></div>
{/if}
