<script lang="ts">
	import { get } from '$lib/api';
	import { relative } from '$lib/format';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import { kindLabel, confidenceLabel, type CandidateSummary } from '$lib/candidates';
	import MagnifyingGlass from 'phosphor-svelte/lib/MagnifyingGlass';
	import WarningCircle from 'phosphor-svelte/lib/WarningCircle';

	let status = $state('');
	let kind = $state('');
	let q = $state('');
	let data = $state<{ candidates: CandidateSummary[]; counts: Record<string, number>; available: boolean } | null>(null);

	const statuses: [string, string][] = [
		['', 'Semua'],
		['draft', 'Draft'],
		['needs_review', 'Perlu tinjauan'],
		['approved', 'Disetujui'],
		['rejected', 'Ditolak'],
		['published', 'Tayang']
	];
	const kinds: [string, string][] = [
		['', 'Semua jenis'],
		['question', 'Soal'],
		['material', 'Materi']
	];

	async function load() {
		data = await get(`/admin/candidates?${new URLSearchParams({ status, kind, q })}`);
	}
	$effect(() => {
		status;
		kind;
		load();
	});
	const total = $derived(data ? Object.values(data.counts).reduce((a, b) => a + b, 0) : 0);
</script>

<svelte:head><title>Antrian review | CMS EduPlay</title></svelte:head>
<div class="flex flex-wrap items-center justify-between gap-4">
	<div>
		<h1 class="text-3xl sm:text-4xl">Antrian review</h1>
		<p class="mt-1 text-ink-soft">Kandidat materi dan soal dari pipeline konten. Setujui untuk menerbitkan soal ke bank soal.</p>
	</div>
</div>

<div class="card mt-6 overflow-hidden">
	<div class="flex flex-wrap items-center gap-3 border-b border-line p-4">
		<div class="flex flex-wrap gap-1 rounded-full bg-paper-deep p-1" role="group" aria-label="Filter status">
			{#each statuses as [k, label]}
				<button class="min-h-9 rounded-full px-3.5 text-sm font-bold transition-colors {status === k ? 'bg-paper text-ink shadow-sm' : 'text-ink-soft hover:text-ink'}" aria-pressed={status === k} onclick={() => (status = k)}>
					{label}{#if data}&nbsp;<span class="text-ink-soft">{k ? (data.counts[k] ?? 0) : total}</span>{/if}
				</button>
			{/each}
		</div>
		<label class="sr-only" for="kind">Jenis</label>
		<select id="kind" class="field !min-h-11 sm:w-44" bind:value={kind}>
			{#each kinds as [k, label]}<option value={k}>{label}</option>{/each}
		</select>
		<form class="relative ml-auto w-full sm:w-72" onsubmit={(e) => { e.preventDefault(); load(); }}>
			<label class="sr-only" for="qs">Cari kandidat</label>
			<MagnifyingGlass size={18} class="pointer-events-none absolute top-1/2 left-3.5 -translate-y-1/2 text-ink-soft" aria-hidden="true" />
			<input id="qs" class="field !min-h-11 pl-10" placeholder="Cari teks atau sumber" bind:value={q} />
		</form>
	</div>

	{#if data && !data.available}
		<p class="px-4 py-12 text-center text-ink-soft">Pipeline konten belum dijalankan di lingkungan ini.</p>
	{:else if data}
		<ul class="divide-y divide-line xl:hidden">
			{#each data.candidates as row (row.kind + row.id)}
				<li>
					<a href="/app/admin/review/{row.kind}/{row.id}" class="block px-4 py-4 hover:bg-paper-deep/40">
						<div class="flex items-start justify-between gap-3">
							<p class="font-bold">{row.label}</p>
							<StatusPill status={row.publication_status ?? row.status} />
						</div>
						<p class="mt-1 text-sm text-ink-soft">{kindLabel(row.kind)} · {row.skill_name || 'belum dipetakan'}{#if row.grade_code} · {row.grade_code}{/if}</p>
						<div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-ink-soft">
							<span class="font-mono">{row.source_code}</span>
							<span>Keyakinan {confidenceLabel(row.confidence)}</span>
							{#if row.blocking_issues}<span class="flex items-center gap-1 text-[#a8264f]"><WarningCircle size={12} weight="fill" aria-hidden="true" />{row.blocking_issues} masalah</span>{/if}
							<span>{relative(row.updated_at)}</span>
						</div>
					</a>
				</li>
			{:else}
				<li class="px-4 py-10 text-center text-ink-soft">Tidak ada kandidat untuk filter ini.</li>
			{/each}
		</ul>
		<div class="hidden overflow-x-auto xl:block">
			<table class="w-full min-w-3xl text-left text-sm">
				<thead class="bg-paper-deep/60 text-xs font-bold text-ink-soft uppercase-none">
					<tr><th class="px-4 py-3">Kandidat</th><th class="px-4">Pemetaan</th><th class="px-4">Keyakinan</th><th class="px-4">QA</th><th class="px-4">Status</th><th class="px-4">Diubah</th></tr>
				</thead>
				<tbody>
					{#each data.candidates as row (row.kind + row.id)}
						<tr class="border-t border-line hover:bg-paper-deep/40">
							<td class="max-w-sm px-4 py-3">
								<a class="font-bold text-ink hover:text-primary-deep" href="/app/admin/review/{row.kind}/{row.id}">{row.label}</a>
								<p class="mt-0.5 text-xs text-ink-soft">{kindLabel(row.kind)} · <span class="font-mono">{row.source_code}</span></p>
							</td>
							<td class="px-4"><p class="font-semibold">{row.skill_name || '—'}</p><p class="text-xs text-ink-soft">{row.grade_code}{row.subject_code ? `, ${row.subject_code}` : ''}</p></td>
							<td class="px-4 whitespace-nowrap">{confidenceLabel(row.confidence)}</td>
							<td class="px-4 whitespace-nowrap">{#if row.blocking_issues}<span class="flex items-center gap-1 font-semibold text-[#a8264f]"><WarningCircle size={13} weight="fill" aria-hidden="true" />{row.blocking_issues}</span>{:else}<span class="text-ink-soft">bersih</span>{/if}</td>
							<td class="px-4"><StatusPill status={row.publication_status ?? row.status} /></td>
							<td class="px-4 whitespace-nowrap text-ink-soft">{relative(row.updated_at)}</td>
						</tr>
					{:else}
						<tr><td colspan="6" class="px-4 py-10 text-center text-ink-soft">Tidak ada kandidat untuk filter ini.</td></tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<div class="skeleton h-64" aria-busy="true"></div>
	{/if}
</div>
