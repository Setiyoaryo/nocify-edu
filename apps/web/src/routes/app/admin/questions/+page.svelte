<script lang="ts">
	import { get } from '$lib/api';
	import { page } from '$app/state';
	import { relative } from '$lib/format';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import Plus from 'phosphor-svelte/lib/Plus';
	import MagnifyingGlass from 'phosphor-svelte/lib/MagnifyingGlass';
	import Eye from 'phosphor-svelte/lib/Eye';

	let status = $state(page.url.searchParams.get('status') ?? '');
	let q = $state('');
	let data = $state<any>(null);
	let types = $state<Record<string, string>>({});
	const statuses = [['', 'Semua'], ['draft', 'Draft'], ['review', 'Review'], ['approved', 'Disetujui'], ['published', 'Tayang'], ['retired', 'Arsip']];
	get('/admin/question-types').then((r) => (types = Object.fromEntries(r.types.map((t: any) => [t.code, t.label]))));

	async function load() {
		data = await get(`/admin/questions?${new URLSearchParams({ status, q })}`);
	}
	$effect(() => {
		status;
		load();
	});
	const total = $derived(data ? Object.values(data.counts as Record<string, number>).reduce((a, b) => a + b, 0) : 0);
</script>

<svelte:head><title>Bank soal | CMS EduPlay</title></svelte:head>
<div class="flex flex-wrap items-center justify-between gap-4">
	<div>
		<h1 class="text-3xl sm:text-4xl">Bank soal</h1>
		<p class="mt-1 text-ink-soft">Setiap soal melewati draft, review, dan persetujuan sebelum tayang.</p>
	</div>
	<a href="/app/admin/questions/new" class="btn btn-primary"><Plus size={18} weight="bold" aria-hidden="true" />Soal baru</a>
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
		<form class="relative ml-auto w-full sm:w-72" onsubmit={(e) => { e.preventDefault(); load(); }}>
			<label class="sr-only" for="qs">Cari soal</label>
			<MagnifyingGlass size={18} class="pointer-events-none absolute top-1/2 left-3.5 -translate-y-1/2 text-ink-soft" aria-hidden="true" />
			<input id="qs" class="field !min-h-11 pl-10" placeholder="Cari kode atau teks soal" bind:value={q} />
		</form>
	</div>
	{#if data}
		<div class="overflow-x-auto">
			<table class="w-full min-w-3xl text-left text-sm">
				<thead class="bg-paper-deep/60 text-xs font-bold text-ink-soft uppercase-none">
					<tr><th class="px-4 py-3">Soal</th><th class="px-4">Skill</th><th class="px-4">Tipe</th><th class="px-4">Kesulitan</th><th class="px-4">Status</th><th class="px-4">Diubah</th></tr>
				</thead>
				<tbody>
					{#each data.questions as row (row.id)}
						<tr class="border-t border-line hover:bg-paper-deep/40">
							<td class="max-w-sm px-4 py-3">
								<a class="font-bold text-ink hover:text-primary-deep" href="/app/admin/questions/{row.id}">{row.prompt}</a>
								<p class="mt-0.5 font-mono text-xs text-ink-soft">{row.code}</p>
							</td>
							<td class="px-4"><p class="font-semibold">{row.skill_name}</p><p class="text-xs text-ink-soft">{row.grade_name}, {row.subject_name}</p></td>
							<td class="px-4 whitespace-nowrap">{types[row.type] ?? row.type}</td>
							<td class="px-4"><span class="flex gap-0.5" aria-label="Kesulitan {row.difficulty} dari 5">{#each [1, 2, 3, 4, 5] as d}<span class="h-2 w-3 rounded-sm {d <= row.difficulty ? 'bg-primary' : 'bg-line'}"></span>{/each}</span></td>
							<td class="px-4"><StatusPill status={row.status} />{#if row.pending}<p class="mt-1 flex items-center gap-1 text-xs text-tangerine-deep"><Eye size={12} weight="bold" aria-hidden="true" />versi lama tayang</p>{/if}</td>
							<td class="px-4 whitespace-nowrap text-ink-soft">{relative(row.updated_at)}</td>
						</tr>
					{:else}
						<tr><td colspan="6" class="px-4 py-10 text-center text-ink-soft">Tidak ada soal untuk filter ini.</td></tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<div class="skeleton h-64" aria-busy="true"></div>
	{/if}
</div>
