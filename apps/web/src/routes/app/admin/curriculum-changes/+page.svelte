<script lang="ts">
	import { get } from '$lib/api';
	import { relative } from '$lib/format';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import { eventStatusLabel, reviewStatusLabel, type ChangeSummary } from '$lib/curriculumChanges';
	import MagnifyingGlass from 'phosphor-svelte/lib/MagnifyingGlass';

	let status = $state('');
	let q = $state('');
	let data = $state<{ changes: ChangeSummary[]; counts: Record<string, number>; available: boolean } | null>(null);

	const statuses: [string, string][] = [
		['', 'Semua'],
		['pending', 'Belum ditinjau'],
		['acknowledged', 'Diterima'],
		['deferred', 'Ditunda'],
		['dismissed', 'Diabaikan']
	];

	async function load() {
		data = await get(`/admin/curriculum-changes?${new URLSearchParams({ status, q })}`);
	}
	$effect(() => {
		status;
		load();
	});
	const total = $derived(data ? Object.values(data.counts).reduce((a, b) => a + b, 0) : 0);
</script>

<svelte:head><title>Perubahan kurikulum | CMS EduPlay</title></svelte:head>
<div class="flex flex-wrap items-center justify-between gap-4">
	<div>
		<h1 class="text-3xl sm:text-4xl">Perubahan kurikulum</h1>
		<p class="mt-1 text-ink-soft">Perubahan dokumen sumber kurikulum yang terdeteksi otomatis, beserta CP dan soal yang terdampak. M7 hanya mendeteksi dan mencatat tinjauan — kurikulum kanonik tidak diubah.</p>
	</div>
</div>

<div class="card mt-6 overflow-hidden">
	<div class="flex flex-wrap items-center gap-3 border-b border-line p-4">
		<div class="flex flex-wrap gap-1 rounded-full bg-paper-deep p-1" role="group" aria-label="Filter status tinjauan">
			{#each statuses as [k, label]}
				<button class="min-h-9 rounded-full px-3.5 text-sm font-bold transition-colors {status === k ? 'bg-paper text-ink shadow-sm' : 'text-ink-soft hover:text-ink'}" aria-pressed={status === k} onclick={() => (status = k)}>
					{label}{#if data}&nbsp;<span class="text-ink-soft">{k ? (data.counts[k] ?? 0) : total}</span>{/if}
				</button>
			{/each}
		</div>
		<form class="relative ml-auto w-full sm:w-72" onsubmit={(e) => { e.preventDefault(); load(); }}>
			<label class="sr-only" for="qs">Cari dokumen</label>
			<MagnifyingGlass size={18} class="pointer-events-none absolute top-1/2 left-3.5 -translate-y-1/2 text-ink-soft" aria-hidden="true" />
			<input id="qs" class="field !min-h-11 pl-10" placeholder="Cari kode dokumen atau URL" bind:value={q} />
		</form>
	</div>

	{#if data && !data.available}
		<p class="px-4 py-12 text-center text-ink-soft">Pipeline konten belum dijalankan di lingkungan ini.</p>
	{:else if data}
		<ul class="divide-y divide-line xl:hidden">
			{#each data.changes as row (row.event_id)}
				<li>
					<a href="/app/admin/curriculum-changes/{row.event_id}" class="block px-4 py-4 hover:bg-paper-deep/40">
						<div class="flex items-start justify-between gap-3">
							<p class="font-mono text-sm font-bold">{row.document_code}</p>
							<StatusPill status={row.review_status} />
						</div>
						<p class="mt-1 text-sm text-ink-soft">{eventStatusLabel(row.event_status)} · {row.affected_questions} soal terdampak</p>
						<div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-ink-soft">
							<span class="text-[#1a6fa3]">+{row.cp_added}</span>
							<span class="text-[#a8264f]">−{row.cp_removed}</span>
							<span>{row.cp_changed} diubah</span>
							<span>{relative(row.created_at)}</span>
						</div>
					</a>
				</li>
			{:else}
				<li class="px-4 py-10 text-center text-ink-soft">Tidak ada perubahan untuk filter ini.</li>
			{/each}
		</ul>
		<div class="hidden overflow-x-auto xl:block">
			<table class="w-full min-w-3xl text-left text-sm">
				<thead class="bg-paper-deep/60 text-xs font-bold text-ink-soft uppercase-none">
					<tr><th class="px-4 py-3">Dokumen</th><th class="px-4">CP berubah</th><th class="px-4">Soal terdampak</th><th class="px-4">Analisis</th><th class="px-4">Tinjauan</th><th class="px-4">Terdeteksi</th></tr>
				</thead>
				<tbody>
					{#each data.changes as row (row.event_id)}
						<tr class="border-t border-line hover:bg-paper-deep/40">
							<td class="max-w-sm px-4 py-3">
								<a class="font-mono font-bold text-ink hover:text-primary-deep" href="/app/admin/curriculum-changes/{row.event_id}">{row.document_code}</a>
								<p class="mt-0.5 truncate text-xs text-ink-soft">{row.url}</p>
							</td>
							<td class="px-4 whitespace-nowrap">
								<span class="font-semibold text-[#1a6fa3]">+{row.cp_added}</span>
								<span class="mx-1 text-ink-soft">/</span>
								<span class="font-semibold">{row.cp_changed}</span>
								<span class="mx-1 text-ink-soft">/</span>
								<span class="font-semibold text-[#a8264f]">−{row.cp_removed}</span>
							</td>
							<td class="px-4 whitespace-nowrap">{row.affected_questions}</td>
							<td class="px-4 whitespace-nowrap">{eventStatusLabel(row.event_status)}</td>
							<td class="px-4"><StatusPill status={row.review_status} /></td>
							<td class="px-4 whitespace-nowrap text-ink-soft">{relative(row.created_at)}</td>
						</tr>
					{:else}
						<tr><td colspan="6" class="px-4 py-10 text-center text-ink-soft">Tidak ada perubahan untuk filter ini.</td></tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<div class="skeleton h-64" aria-busy="true"></div>
	{/if}
</div>
