<script lang="ts">
	import Check from 'phosphor-svelte/lib/Check';
	import CircleNotch from 'phosphor-svelte/lib/CircleNotch';
	import Clock from 'phosphor-svelte/lib/Clock';
	import WarningCircle from 'phosphor-svelte/lib/WarningCircle';
	import X from 'phosphor-svelte/lib/X';
	import HourglassHigh from 'phosphor-svelte/lib/HourglassHigh';
	import {
		calculateJobProgress,
		formatDuration,
		getJobStages,
		getStageStepStatus,
		stageLabel,
		stageDescription,
		type PipelineJob
	} from '$lib/pipeline';

	let { job }: { job: PipelineJob } = $props();

	const stages = $derived(getJobStages(job));
	const progress = $derived(calculateJobProgress(job));
	const duration = $derived(formatDuration(job.started_at, job.finished_at));
</script>

<div class="card p-5 sm:p-6" data-testid="pipeline-step-tracker">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div>
			<div class="flex items-center gap-2">
				<span class="text-xs font-bold uppercase tracking-wider text-ink-soft">
					Progres Pipeline
				</span>
				{#if job.status === 'running'}
					<span class="inline-flex items-center gap-1.5 rounded-full bg-sky-tint px-2.5 py-0.5 text-xs font-bold text-[#1a6fa3]">
						<CircleNotch size={12} class="animate-spin" aria-hidden="true" />
						Sedang Berjalan
					</span>
				{:else if job.status === 'done'}
					<span class="inline-flex items-center gap-1.5 rounded-full bg-leaf-tint px-2.5 py-0.5 text-xs font-bold text-leaf-deep">
						<Check size={12} weight="bold" aria-hidden="true" />
						Selesai
					</span>
				{:else if job.status === 'failed'}
					<span class="inline-flex items-center gap-1.5 rounded-full bg-rose-tint px-2.5 py-0.5 text-xs font-bold text-[#a8264f]">
						<WarningCircle size={12} weight="bold" aria-hidden="true" />
						Gagal
					</span>
				{:else if job.status === 'cancelled'}
					<span class="inline-flex items-center gap-1.5 rounded-full bg-paper-deep px-2.5 py-0.5 text-xs font-bold text-ink-soft">
						<X size={12} weight="bold" aria-hidden="true" />
						Dibatalkan
					</span>
				{:else}
					<span class="inline-flex items-center gap-1.5 rounded-full bg-paper-deep px-2.5 py-0.5 text-xs font-bold text-ink-soft">
						<Clock size={12} aria-hidden="true" />
						Menunggu Antrean
					</span>
				{/if}
			</div>

			<h3 class="mt-1 text-xl font-bold">
				{#if job.status === 'running'}
					Tahap {progress.stepNumber} dari {progress.totalCount}: {progress.currentStageLabel}
				{:else if job.status === 'done'}
					Selesai — Semua {progress.totalCount} Tahapan Berhasil
				{:else if job.status === 'queued'}
					Menunggu giliran eksekusi worker
				{:else if job.status === 'failed'}
					Eksekusi terhenti di tahap: {progress.currentStageLabel || 'awal'}
				{:else}
					Job dibatalkan
				{/if}
			</h3>
			{#if progress.currentStageDesc}
				<p class="mt-0.5 text-sm text-ink-soft">{progress.currentStageDesc}</p>
			{/if}
		</div>

		<div class="flex items-center gap-4 text-right">
			<div>
				<div class="text-xs text-ink-soft">Durasi</div>
				<div class="font-mono text-sm font-bold text-ink">{duration}</div>
			</div>
			<div>
				<div class="text-xs text-ink-soft">Penyelesaian</div>
				<div class="font-mono text-lg font-black text-primary-deep">{progress.percent}%</div>
			</div>
		</div>
	</div>

	<!-- Progress Bar Visual -->
	<div class="mt-4">
		<div class="h-3 w-full overflow-hidden rounded-full bg-paper-deep p-0.5" role="progressbar" aria-valuenow={progress.percent} aria-valuemin="0" aria-valuemax="100">
			<div
				class="relative h-full rounded-full transition-all duration-500 ease-out {job.status === 'failed' ? 'bg-[#a8264f]' : job.status === 'cancelled' ? 'bg-ink-soft' : job.status === 'done' ? 'bg-leaf-deep' : 'bg-primary'}"
				style="width: {progress.percent}%;"
			>
				{#if job.status === 'running'}
					<div class="h-full w-full animate-pulse bg-white/25 rounded-full" aria-hidden="true"></div>
				{/if}
			</div>
		</div>
	</div>

	<!-- Step Flow Indicators -->
	<div class="mt-6 overflow-x-auto pb-2">
		<ol class="flex min-w-[540px] items-start justify-between gap-2" aria-label="Alur tahapan pipeline">
			{#each stages as stage, idx (stage)}
				{@const status = getStageStepStatus(job, stage)}
				{@const isLast = idx === stages.length - 1}
				<li class="relative flex flex-1 flex-col items-center text-center">
					<!-- Connector line to next step -->
					{#if !isLast}
						<div
							class="absolute top-4 left-1/2 -z-0 h-0.5 w-full -translate-y-1/2 transition-colors {status === 'done' ? 'bg-leaf-deep/60' : 'bg-line'}"
							aria-hidden="true"
						></div>
					{/if}

					<!-- Step Icon Node -->
					<div
						class="relative z-10 flex size-8 items-center justify-center rounded-full text-xs font-bold transition-all shadow-xs
						{status === 'done' ? 'bg-leaf-deep text-white ring-4 ring-leaf-tint' :
						 status === 'running' ? 'bg-primary text-white ring-4 ring-primary-tint animate-pulse' :
						 status === 'failed' ? 'bg-[#a8264f] text-white ring-4 ring-rose-tint' :
						 status === 'cancelled' ? 'bg-paper-deep text-ink-soft' :
						 'bg-paper border-2 border-line text-ink-soft'}"
					>
						{#if status === 'done'}
							<Check size={16} weight="bold" aria-hidden="true" />
						{:else if status === 'running'}
							<CircleNotch size={16} class="animate-spin" aria-hidden="true" />
						{:else if status === 'failed'}
							<WarningCircle size={16} weight="bold" aria-hidden="true" />
						{:else if status === 'cancelled'}
							<X size={14} weight="bold" aria-hidden="true" />
						{:else}
							<span>{idx + 1}</span>
						{/if}
					</div>

					<!-- Step Label and Context -->
					<div class="mt-2.5 px-1">
						<p class="text-xs font-bold leading-tight {status === 'running' ? 'text-primary-deep' : status === 'done' ? 'text-ink' : 'text-ink-soft'}">
							{stageLabel[stage] ?? stage}
						</p>
						<p class="mt-0.5 line-clamp-1 text-[11px] text-ink-soft">
							{#if status === 'done'}
								Selesai
							{:else if status === 'running'}
								Sedang proses
							{:else if status === 'failed'}
								Gagal
							{:else}
								Menunggu
							{/if}
						</p>
					</div>
				</li>
			{/each}
		</ol>
	</div>
</div>
