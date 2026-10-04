<script lang="ts">
	import { get, post, ApiError } from '$lib/api';
	import { session } from '$lib/session.svelte';
	import { dateTime } from '$lib/format';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import {
		hasActiveJob,
		jobKindLabel,
		jobProgressLabel,
		frequencyLabel,
		sourceStatusLabel,
		CRAWL_FREQUENCIES,
		type PipelineJob,
		type PipelineSource
	} from '$lib/pipeline';
	import { onDestroy } from 'svelte';
	import Play from 'phosphor-svelte/lib/Play';
	import ArrowClockwise from 'phosphor-svelte/lib/ArrowClockwise';
	import RocketLaunch from 'phosphor-svelte/lib/RocketLaunch';

	let sources = $state<PipelineSource[]>([]);
	let jobs = $state<PipelineJob[]>([]);
	let available = $state(true);
	let loaded = $state(false);
	let err = $state('');
	let msg = $state('');
	let busy = $state(false);
	let timer: ReturnType<typeof setTimeout>;
	const isReviewer = $derived(session.user?.role === 'reviewer' || session.user?.role === 'admin');

	async function load() {
		clearTimeout(timer);
		try {
			const [s, j] = await Promise.all([
				get('/admin/pipeline/sources'),
				get('/admin/pipeline/jobs')
			]);
			sources = s.sources;
			available = s.available;
			jobs = j.jobs;
			err = '';
		} catch (e) {
			err = e instanceof ApiError ? e.message : 'Gagal memuat data pipeline.';
		} finally {
			loaded = true;
		}
		if (hasActiveJob(jobs)) timer = setTimeout(load, 3000);
	}
	load();
	onDestroy(() => clearTimeout(timer));

	async function run(kind: string, params: Record<string, any>, note: string) {
		busy = true;
		msg = err = '';
		try {
			await post('/admin/pipeline/jobs', { kind, params });
			msg = note;
			await load();
		} catch (e) {
			err = e instanceof ApiError ? e.message : 'Gagal menjalankan.';
		} finally {
			busy = false;
		}
	}
	async function sourceStatus(code: string, body: Record<string, any>) {
		err = '';
		try {
			await post(`/admin/pipeline/sources/${code}/status`, body);
			await load();
		} catch (e) {
			err = e instanceof ApiError ? e.message : 'Gagal mengubah sumber.';
		}
	}
	async function jobAction(id: number, action: 'cancel' | 'retry') {
		err = '';
		try {
			await post(`/admin/pipeline/jobs/${id}/${action}`);
			await load();
		} catch (e) {
			err = e instanceof ApiError ? e.message : 'Gagal.';
		}
	}
	async function publishAll() {
		busy = true;
		msg = err = '';
		try {
			const r = await post('/admin/pipeline/publish', { all: true });
			msg = `Terbit: ${r.report.published}, tertaut: ${r.report.linked}, gagal: ${r.report.failed}, sisa: ${r.report.remaining}.`;
			await load();
		} catch (e) {
			err = e instanceof ApiError ? e.message : 'Gagal menerbitkan.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Pipeline | CMS</title></svelte:head>
<h1 class="text-3xl sm:text-4xl">Pipeline konten</h1>
<p class="mt-1 max-w-3xl text-ink-soft">
	Jalankan dan pantau pipeline dari sini: crawl sumber, ekstrak, petakan ke kurikulum, lalu bangun
	kandidat soal. Reviewer menyetujui di antrian review, lalu menerbitkan ke bank soal.
</p>

<div class="mt-6 flex flex-wrap gap-2">
	<button class="btn btn-primary" disabled={busy || !available} onclick={() => run('run_pipeline', {}, 'Pipeline dijalankan.')}>
		<Play size={18} weight="fill" aria-hidden="true" />Jalankan pipeline
	</button>
	{#if isReviewer}
		<button class="btn btn-quiet" disabled={busy || !available} onclick={publishAll}>
			<RocketLaunch size={18} weight="bold" aria-hidden="true" />Terbitkan semua yang disetujui
		</button>
	{/if}
	<button class="btn btn-quiet" disabled={busy} onclick={load}>
		<ArrowClockwise size={18} weight="bold" aria-hidden="true" />Muat ulang
	</button>
</div>
{#if err}<p class="field-error mt-3" role="alert">{err}</p>{/if}
{#if msg}<p class="mt-3 font-semibold text-leaf-deep" role="status">{msg}</p>{/if}

{#if !available && loaded}
	<div class="card mt-6 p-6" role="status">
		<p class="font-semibold">Pipeline belum siap di lingkungan ini.</p>
		<p class="mt-1 text-sm text-ink-soft">Worker pipeline belum membuat skema staging. Jalankan layanan pipeline lebih dulu.</p>
	</div>
{/if}

<section class="mt-8" aria-labelledby="src">
	<h2 id="src" class="text-2xl">Sumber</h2>
	<p class="mt-1 text-sm text-ink-soft">Sumber aktif dengan jadwal (harian/mingguan/bulanan) dicrawl otomatis oleh worker.</p>
	<div class="card mt-4 overflow-x-auto">
		<table class="w-full min-w-3xl text-left text-sm">
			<thead class="bg-paper-deep/60 text-xs font-bold text-ink-soft">
				<tr><th class="px-4 py-3">Sumber</th><th>Status</th><th>Aktif</th><th>Jadwal</th><th>Terakhir crawl</th><th></th></tr>
			</thead>
			<tbody>
				{#each sources as s (s.code)}
					<tr class="border-t border-line align-top [&>td]:px-4 [&>td]:py-3">
						<td><b>{s.name}</b><br /><span class="font-mono text-xs text-ink-soft">{s.code}</span></td>
						<td>{sourceStatusLabel[s.status] ?? s.status}</td>
						<td>
							<label class="flex min-h-11 items-center gap-2">
								<input type="checkbox" class="size-5 accent-primary" checked={s.enabled} disabled={!available}
									onchange={() => sourceStatus(s.code, { enabled: !s.enabled })} />
								<span class="text-ink-soft">{s.enabled ? 'aktif' : 'mati'}</span>
							</label>
						</td>
						<td>
							<select class="field !min-h-11 w-36" value={s.crawl_frequency} disabled={!available}
								onchange={(e) => sourceStatus(s.code, { crawl_frequency: e.currentTarget.value })}>
								{#each CRAWL_FREQUENCIES as f}<option value={f}>{frequencyLabel[f]}</option>{/each}
							</select>
						</td>
						<td>{s.last_crawled_at ? dateTime(s.last_crawled_at) : '—'}</td>
						<td class="text-right">
							{#if s.pending_jobs > 0}
								<span class="pill bg-sky-tint text-[#1a6fa3]">{s.pending_jobs} menunggu</span>
							{:else}
								<button class="btn btn-quiet btn-sm whitespace-nowrap" disabled={busy || !available}
									onclick={() => run('run_pipeline', { sources: [s.code] }, `Menjalankan ${s.code}.`)}>Jalankan sekarang</button>
							{/if}
						</td>
					</tr>
				{:else}
					<tr><td colspan="6" class="px-4 py-8 text-ink-soft">{loaded ? 'Belum ada sumber terdaftar.' : 'Memuat…'}</td></tr>
				{/each}
			</tbody>
		</table>
	</div>
</section>

<section class="mt-10" aria-labelledby="jobs">
	<h2 id="jobs" class="text-2xl">Riwayat job</h2>
	<div class="card mt-4 overflow-x-auto">
		<table class="w-full min-w-3xl text-left text-sm">
			<thead class="bg-paper-deep/60 text-xs font-bold text-ink-soft">
				<tr><th class="px-4 py-3">#</th><th>Jenis</th><th>Status</th><th>Progres</th><th>Dibuat</th><th></th></tr>
			</thead>
			<tbody>
				{#each jobs as j (j.id)}
					<tr class="border-t border-line align-top [&>td]:px-4 [&>td]:py-3">
						<td><a class="font-mono text-primary-deep hover:underline" href="/app/admin/pipeline/{j.id}">{j.id}</a></td>
						<td>{jobKindLabel[j.kind] ?? j.kind}</td>
						<td><StatusPill status={j.status} /></td>
						<td>
							{jobProgressLabel(j)}
							{#if j.error}<p class="mt-1 text-xs text-[#a8264f]">{j.error}</p>{/if}
						</td>
						<td class="text-ink-soft">{dateTime(j.created_at)}</td>
						<td class="space-y-1 text-right">
							{#if j.status === 'queued' || j.status === 'running'}
								<button class="btn btn-sm text-error" onclick={() => jobAction(j.id, 'cancel')}>Batalkan</button>
							{:else if j.status === 'failed' || j.status === 'cancelled'}
								{#if j.attempts < j.max_attempts}
									<button class="btn btn-quiet btn-sm" onclick={() => jobAction(j.id, 'retry')}>Ulangi</button>
								{/if}
							{/if}
						</td>
					</tr>
				{:else}
					<tr><td colspan="6" class="px-4 py-8 text-ink-soft">{loaded ? 'Belum ada job.' : 'Memuat…'}</td></tr>
				{/each}
			</tbody>
		</table>
	</div>
</section>
