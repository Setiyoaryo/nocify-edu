<script lang="ts">
	import Activity from 'phosphor-svelte/lib/Pulse';
	import Cpu from 'phosphor-svelte/lib/Cpu';
	import Clock from 'phosphor-svelte/lib/Clock';
	import WarningCircle from 'phosphor-svelte/lib/WarningCircle';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	import { dateTime, relative } from '$lib/format';
	import { workerStatusStyle, type WorkerInfo } from '$lib/pipeline';

	let { worker }: { worker: WorkerInfo | null } = $props();

	const status = $derived(worker?.status ?? 'stopped');
	const style = $derived(workerStatusStyle[status] ?? workerStatusStyle.stopped);
	const label = $derived(worker?.label ?? style.label);
</script>

<div class="card p-4 sm:p-5" data-testid="worker-health-indicator">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div class="flex items-center gap-3">
			<div class="flex size-10 shrink-0 items-center justify-center rounded-xl bg-paper-deep text-ink">
				<Cpu size={22} aria-hidden="true" />
			</div>
			<div>
				<div class="flex items-center gap-2">
					<span class="text-xs font-bold uppercase tracking-wider text-ink-soft">
						Status Worker Scraper & Pipeline
					</span>
					<!-- Live Status Pill -->
					<span class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-bold {style.badge}">
						<span class="size-2 rounded-full {style.dot}"></span>
						{label}
					</span>
				</div>
				<h4 class="mt-0.5 text-sm sm:text-base font-bold text-ink">
					{#if status === 'active'}
						Worker Berjalan Otomatis (Memproses)
					{:else if status === 'idle'}
						Worker Berjalan Otomatis (Siaga / Idle)
					{:else if status === 'stalled'}
						Worker Terhenti / Memerlukan Perhatian
					{:else}
						Worker Belum Aktif / Offline
					{/if}
				</h4>
			</div>
		</div>

		<!-- Heartbeat Timestamp -->
		<div class="flex items-center gap-4 text-xs text-ink-soft">
			<div class="text-right">
				<div class="flex items-center justify-end gap-1">
					<Activity size={14} class={status === 'active' ? 'text-leaf-deep animate-pulse' : 'text-ink-soft'} aria-hidden="true" />
					<span class="font-semibold">Detak Jantung (Heartbeat):</span>
				</div>
				<div class="mt-0.5 font-mono text-ink">
					{#if worker?.last_heartbeat_at}
						{dateTime(worker.last_heartbeat_at)} ({relative(worker.last_heartbeat_at)})
					{:else}
						Belum ada sinyal
					{/if}
				</div>
			</div>
		</div>
	</div>

	<!-- Detail / Context Message -->
	<div class="mt-3 rounded-xl border border-line bg-paper/60 px-3.5 py-2.5 text-xs flex flex-wrap items-center justify-between gap-2">
		<div class="flex items-center gap-2 text-ink">
			{#if status === 'active'}
				<CheckCircle size={16} weight="fill" class="text-leaf-deep shrink-0" aria-hidden="true" />
			{:else if status === 'idle'}
				<Clock size={16} class="text-[#1a6fa3] shrink-0" aria-hidden="true" />
			{:else}
				<WarningCircle size={16} weight="fill" class="text-[#a8264f] shrink-0" aria-hidden="true" />
			{/if}
			<span>{worker?.message ?? (status === 'stopped' ? 'Worker scheduler pipeline sedang offline. Pastikan layanan worker dijalankan.' : 'Worker scheduler berjalan otomatis di latar belakang.')}</span>
		</div>

		<div class="flex items-center gap-3 text-ink-soft font-mono">
			<span class="pill text-xs font-semibold {status === 'active' || status === 'idle' ? 'bg-leaf-tint text-leaf-deep' : 'bg-rose-tint text-[#a8264f]'}">
				{status === 'active' || status === 'idle' ? 'Otomatisasi: Aktif' : 'Otomatisasi: Mati'}
			</span>
			<span>•</span>
			<span>Sedang jalan: <b class="text-ink">{worker?.running_jobs ?? 0}</b></span>
			<span>•</span>
			<span>Antrean: <b class="text-ink">{worker?.queued_jobs ?? 0}</b></span>
		</div>
	</div>
</div>
