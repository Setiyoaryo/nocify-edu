<script lang="ts">
	import { get, ApiError } from '$lib/api';
	import { page } from '$app/state';
	import { dateTime } from '$lib/format';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import { isActive, jobKindLabel, stageLabel, type PipelineJob } from '$lib/pipeline';
	import { onDestroy } from 'svelte';

	const id = page.params.id;
	let job = $state<PipelineJob | null>(null);
	let loadErr = $state('');
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

	const stages = $derived(Object.entries(job?.progress?.stages ?? {}));
</script>

<svelte:head><title>Job {id} | CMS</title></svelte:head>
<a href="/app/admin/pipeline" class="text-sm font-bold text-primary-deep hover:underline">Kembali ke pipeline</a>

{#if job}
	<div class="mt-2 flex flex-wrap items-center gap-3">
		<h1 class="text-3xl">Job #{job.id}</h1>
		<StatusPill status={job.status} />
		<span class="text-sm text-ink-soft">{jobKindLabel[job.kind] ?? job.kind}</span>
	</div>

	<div class="mt-6 grid gap-6 lg:grid-cols-2">
		<section class="card p-5 sm:p-6" aria-labelledby="meta">
			<h2 id="meta" class="text-xl">Ringkasan</h2>
			<dl class="mt-3 space-y-2 text-sm">
				<div class="flex gap-4"><dt class="w-32 text-ink-soft">Percobaan</dt><dd>{job.attempts}/{job.max_attempts}</dd></div>
				<div class="flex gap-4"><dt class="w-32 text-ink-soft">Dibuat</dt><dd>{dateTime(job.created_at)}</dd></div>
				<div class="flex gap-4"><dt class="w-32 text-ink-soft">Mulai</dt><dd>{job.started_at ? dateTime(job.started_at) : '—'}</dd></div>
				<div class="flex gap-4"><dt class="w-32 text-ink-soft">Selesai</dt><dd>{job.finished_at ? dateTime(job.finished_at) : '—'}</dd></div>
			</dl>
			{#if job.error}<p class="mt-3 rounded-xl bg-rose-tint p-3 text-sm text-[#a8264f]" role="alert">{job.error}</p>{/if}
			<details class="mt-4">
				<summary class="cursor-pointer text-sm font-bold text-primary-deep">Parameter</summary>
				<pre class="mt-2 overflow-x-auto rounded-xl bg-night p-3 text-xs text-white">{JSON.stringify(job.params, null, 2)}</pre>
			</details>
		</section>

		<section class="card p-5 sm:p-6" aria-labelledby="stages">
			<h2 id="stages" class="text-xl">Tahapan</h2>
			{#if stages.length}
				<ul class="mt-3 space-y-3">
					{#each stages as [name, result]}
						<li class="rounded-xl bg-paper-deep/50 p-3">
							<p class="font-bold">{stageLabel[name] ?? name}</p>
							<pre class="mt-1 overflow-x-auto text-xs text-ink-soft">{JSON.stringify(result, null, 2)}</pre>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="mt-3 text-sm text-ink-soft">Belum ada tahap yang selesai.</p>
			{/if}
		</section>
	</div>
{:else if loadErr}
	<div class="card mt-6 p-6" role="alert"><p class="font-semibold text-[#a8264f]">{loadErr}</p></div>
{:else}
	<div class="skeleton mt-6 h-96" aria-busy="true"></div>
{/if}
