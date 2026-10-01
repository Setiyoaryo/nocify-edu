<script lang="ts">
	import { get, post, ApiError } from '$lib/api';
	import { session } from '$lib/session.svelte';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import {
		completionRate,
		kindLabel,
		qualityLabel,
		shareLabel,
		type CreatorSummary
	} from '$lib/creators';
	import MagnifyingGlass from 'phosphor-svelte/lib/MagnifyingGlass';
	import UserPlus from 'phosphor-svelte/lib/UserPlus';

	let status = $state('');
	let q = $state('');
	let data = $state<{ creators: CreatorSummary[]; counts: Record<string, number> } | null>(null);
	let showForm = $state(false);
	let form = $state({ display_name: '', code: '', kind: 'external', share_pct: '' });
	let formErr = $state('');
	let formMsg = $state('');
	let busy = $state(false);

	const statuses: [string, string][] = [
		['', 'Semua'],
		['active', 'Aktif'],
		['inactive', 'Nonaktif']
	];
	const isEditor = $derived(session.user?.role === 'content_editor' || session.user?.role === 'admin');

	async function load() {
		data = await get(`/admin/creators?${new URLSearchParams({ status, q })}`);
	}
	$effect(() => {
		status;
		load();
	});
	const total = $derived(data ? Object.values(data.counts).reduce((a, b) => a + b, 0) : 0);

	async function create(e: Event) {
		e.preventDefault();
		busy = true;
		formErr = formMsg = '';
		try {
			const body: Record<string, unknown> = { display_name: form.display_name, kind: form.kind };
			if (form.code) body.code = form.code;
			if (form.share_pct !== '') body.share_pct = Number(form.share_pct);
			await post('/admin/creators', body);
			form = { display_name: '', code: '', kind: 'external', share_pct: '' };
			showForm = false;
			formMsg = 'Kreator ditambahkan.';
			await load();
		} catch (err) {
			formErr =
				err instanceof ApiError
					? (Object.values(err.fields ?? {})[0] ?? err.message)
					: 'Gagal menambah kreator.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Kreator | CMS EduPlay</title></svelte:head>
<div class="flex flex-wrap items-center justify-between gap-4">
	<div>
		<h1 class="text-3xl sm:text-4xl">Kreator</h1>
		<p class="mt-1 text-ink-soft">Atribusi penulis konten dan performanya: jumlah soal, penggunaan, dan skor kualitas dari hasil belajar anak. M8 hanya mencatat atribusi — belum ada perhitungan bagi hasil.</p>
	</div>
	{#if isEditor}
		<button class="btn btn-primary" onclick={() => (showForm = !showForm)}><UserPlus size={18} weight="bold" aria-hidden="true" />Tambah kreator</button>
	{/if}
</div>

{#if showForm}
	<form class="card mt-4 grid gap-3 p-4 sm:grid-cols-4" onsubmit={create}>
		<label class="sm:col-span-2"><span class="label">Nama</span><input class="field" required bind:value={form.display_name} /></label>
		<label><span class="label">Kode (opsional)</span><input class="field" placeholder="otomatis" bind:value={form.code} /></label>
		<label><span class="label">Jenis</span>
			<select class="field" bind:value={form.kind}>
				<option value="external">Eksternal</option>
				<option value="staff">Internal</option>
			</select>
		</label>
		<label><span class="label">Bagi hasil (%)</span><input class="field" type="number" min="0" max="100" step="0.5" bind:value={form.share_pct} /></label>
		<div class="flex items-end gap-2 sm:col-span-3">
			<button class="btn btn-primary" disabled={busy}>Simpan</button>
			<button type="button" class="btn btn-quiet" onclick={() => (showForm = false)}>Batal</button>
		</div>
		{#if formErr}<p class="field-error sm:col-span-4" role="alert">{formErr}</p>{/if}
	</form>
{/if}
{#if formMsg}<p class="mt-3 font-semibold text-leaf-deep" role="status">{formMsg}</p>{/if}

<div class="card mt-6 overflow-hidden">
	<div class="flex flex-wrap items-center gap-3 border-b border-line p-4">
		<div class="flex flex-wrap gap-1 rounded-full bg-paper-deep p-1" role="group" aria-label="Filter status kreator">
			{#each statuses as [k, label]}
				<button class="min-h-9 rounded-full px-3.5 text-sm font-bold transition-colors {status === k ? 'bg-paper text-ink shadow-sm' : 'text-ink-soft hover:text-ink'}" aria-pressed={status === k} onclick={() => (status = k)}>
					{label}{#if data}&nbsp;<span class="text-ink-soft">{k ? (data.counts[k] ?? 0) : total}</span>{/if}
				</button>
			{/each}
		</div>
		<form class="relative ml-auto w-full sm:w-72" onsubmit={(e) => { e.preventDefault(); load(); }}>
			<label class="sr-only" for="qc">Cari kreator</label>
			<MagnifyingGlass size={18} class="pointer-events-none absolute top-1/2 left-3.5 -translate-y-1/2 text-ink-soft" aria-hidden="true" />
			<input id="qc" class="field !min-h-11 pl-10" placeholder="Cari nama atau kode" bind:value={q} />
		</form>
	</div>

	{#if data}
		<ul class="divide-y divide-line xl:hidden">
			{#each data.creators as row (row.id)}
				<li>
					<a href="/app/admin/creators/{row.id}" class="block px-4 py-4 hover:bg-paper-deep/40">
						<div class="flex items-start justify-between gap-3">
							<div>
								<p class="font-bold">{row.display_name}</p>
								<p class="font-mono text-xs text-ink-soft">{row.code}</p>
							</div>
							<StatusPill status={row.status} />
						</div>
						<p class="mt-1 text-sm text-ink-soft">{kindLabel(row.kind)} · {row.questions} soal · {shareLabel(row.share_pct)}</p>
						<div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-ink-soft">
							<span>{row.usage_count} dipakai</span>
							<span>{completionRate(row.usage_count, row.completion_count)}% selesai</span>
							<span>{qualityLabel(row.avg_quality)}</span>
						</div>
					</a>
				</li>
			{:else}
				<li class="px-4 py-10 text-center text-ink-soft">Belum ada kreator untuk filter ini.</li>
			{/each}
		</ul>
		<div class="hidden overflow-x-auto xl:block">
			<table class="w-full min-w-3xl text-left text-sm">
				<thead class="bg-paper-deep/60 text-xs font-bold text-ink-soft uppercase-none">
					<tr><th class="px-4 py-3">Kreator</th><th class="px-4">Jenis</th><th class="px-4">Soal</th><th class="px-4">Dipakai</th><th class="px-4">Selesai</th><th class="px-4">Kualitas</th><th class="px-4">Bagi hasil</th><th class="px-4">Status</th></tr>
				</thead>
				<tbody>
					{#each data.creators as row (row.id)}
						<tr class="border-t border-line hover:bg-paper-deep/40">
							<td class="px-4 py-3">
								<a class="font-bold text-ink hover:text-primary-deep" href="/app/admin/creators/{row.id}">{row.display_name}</a>
								<p class="font-mono text-xs text-ink-soft">{row.code}</p>
							</td>
							<td class="px-4 whitespace-nowrap">{kindLabel(row.kind)}</td>
							<td class="px-4 whitespace-nowrap">{row.questions}{#if row.published}<span class="text-ink-soft"> · {row.published} tayang</span>{/if}</td>
							<td class="px-4 whitespace-nowrap">{row.usage_count}</td>
							<td class="px-4 whitespace-nowrap">{completionRate(row.usage_count, row.completion_count)}%</td>
							<td class="px-4 whitespace-nowrap">{qualityLabel(row.avg_quality)}{#if row.avg_quality}<span class="text-ink-soft"> · {row.avg_quality}</span>{/if}</td>
							<td class="px-4 whitespace-nowrap">{shareLabel(row.share_pct)}</td>
							<td class="px-4"><StatusPill status={row.status} /></td>
						</tr>
					{:else}
						<tr><td colspan="8" class="px-4 py-10 text-center text-ink-soft">Belum ada kreator untuk filter ini.</td></tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<div class="skeleton h-64" aria-busy="true"></div>
	{/if}
</div>
