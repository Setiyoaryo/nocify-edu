<script lang="ts">
	import { get, patch, put, del, ApiError } from '$lib/api';
	import { page } from '$app/state';
	import { session } from '$lib/session.svelte';
	import { dateTime } from '$lib/format';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import {
		completionRate,
		kindLabel,
		qualityLabel,
		shareLabel,
		type CreatorDetail
	} from '$lib/creators';

	const id = page.params.id;

	let data = $state<CreatorDetail | null>(null);
	let loadErr = $state('');
	let share = $state('');
	let status = $state('active');
	let saveMsg = $state('');
	let saveErr = $state('');
	let busy = $state(false);
	let assignID = $state('');
	let assignMsg = $state('');
	let assignErr = $state('');

	const isEditor = $derived(session.user?.role === 'content_editor' || session.user?.role === 'admin');

	async function load() {
		loadErr = '';
		try {
			const r = await get(`/admin/creators/${id}`);
			data = r;
			share = String(r.creator.share_pct);
			status = r.creator.status;
		} catch (e) {
			loadErr =
				e instanceof ApiError && e.status === 404
					? 'Kreator tidak ditemukan.'
					: e instanceof ApiError
						? e.message
						: 'Gagal memuat kreator.';
		}
	}
	load();

	async function save() {
		busy = true;
		saveMsg = saveErr = '';
		try {
			const r = await patch(`/admin/creators/${id}`, { share_pct: Number(share), status });
			data = { ...data!, creator: { ...data!.creator, ...r.creator } };
			saveMsg = 'Perubahan disimpan.';
		} catch (e) {
			saveErr = e instanceof ApiError ? (Object.values(e.fields ?? {})[0] ?? e.message) : 'Gagal menyimpan.';
		} finally {
			busy = false;
		}
	}

	async function assign(e: Event) {
		e.preventDefault();
		assignMsg = assignErr = '';
		try {
			await put(`/admin/questions/${assignID.trim()}/creator`, { creator_id: id });
			assignID = '';
			assignMsg = 'Soal ditautkan ke kreator ini.';
			await load();
		} catch (err) {
			assignErr = err instanceof ApiError ? (Object.values(err.fields ?? {})[0] ?? err.message) : 'Gagal menautkan soal.';
		}
	}

	async function unassign(questionID: string) {
		assignMsg = assignErr = '';
		try {
			await del(`/admin/questions/${questionID}/creator`);
			assignMsg = 'Atribusi dilepas.';
			await load();
		} catch (err) {
			assignErr = err instanceof ApiError ? err.message : 'Gagal melepas atribusi.';
		}
	}
</script>

<svelte:head><title>{data ? data.creator.display_name : 'Kreator'} | CMS</title></svelte:head>
<a href="/app/admin/creators" class="text-sm font-bold text-primary-deep hover:underline">Kembali ke daftar kreator</a>

{#if data}
	<div class="mt-2 flex flex-wrap items-center gap-3">
		<h1 class="text-2xl sm:text-3xl">{data.creator.display_name}</h1>
		<StatusPill status={data.creator.status} />
		<span class="pill bg-paper-deep text-ink-soft">{kindLabel(data.creator.kind)}</span>
		<span class="pill bg-primary-tint text-primary-deep">Bagi hasil {shareLabel(data.creator.share_pct)}</span>
	</div>
	<p class="mt-1 font-mono text-sm text-ink-soft">{data.creator.code}</p>

	<div class="mt-6 grid gap-6 lg:grid-cols-[1.6fr_1fr]">
		<div class="space-y-6">
			<section class="card p-5 sm:p-6" aria-labelledby="qs">
				<div class="flex flex-wrap items-center justify-between gap-3">
					<h2 id="qs" class="text-xl">Soal kreator</h2>
					<p class="text-sm text-ink-soft">{data.creator.questions} soal · {data.creator.usage_count} dipakai · {data.creator.completion_count} selesai</p>
				</div>
				<div class="mt-4 overflow-x-auto">
					<table class="w-full min-w-2xl text-left text-sm">
						<thead class="bg-paper-deep/60 text-xs font-bold text-ink-soft uppercase-none">
							<tr><th class="px-3 py-2">Soal</th><th class="px-3">Skill</th><th class="px-3">Dipakai</th><th class="px-3">Selesai</th><th class="px-3">Kualitas</th>{#if isEditor}<th class="px-3"><span class="sr-only">Aksi</span></th>{/if}</tr>
						</thead>
						<tbody>
							{#each data.questions as row (row.question_id)}
								<tr class="border-t border-line">
									<td class="px-3 py-2">
										<a class="font-mono font-bold text-ink hover:text-primary-deep" href="/app/admin/questions/{row.question_id}">{row.code}</a>
										<p class="text-xs text-ink-soft">{row.type} · {row.status}</p>
									</td>
									<td class="px-3 text-ink-soft">{row.skill_name || `skill #${row.skill_id}`}</td>
									<td class="px-3 whitespace-nowrap">{row.usage_count}</td>
									<td class="px-3 whitespace-nowrap">{completionRate(row.usage_count, row.completion_count)}%</td>
									<td class="px-3 whitespace-nowrap">{qualityLabel(row.quality_score)}{#if row.quality_score}<span class="text-ink-soft"> · {row.quality_score}</span>{/if}</td>
									{#if isEditor}
										<td class="px-3 text-right"><button class="btn btn-quiet !min-h-8 !px-2 text-xs" onclick={() => unassign(row.question_id)}>Lepas</button></td>
									{/if}
								</tr>
							{:else}
								<tr><td colspan={isEditor ? 6 : 5} class="px-3 py-8 text-center text-ink-soft">Belum ada soal yang ditautkan ke kreator ini.</td></tr>
							{/each}
						</tbody>
					</table>
				</div>
				{#if data.questions.length && data.questions[0].computed_at}
					<p class="mt-3 text-xs text-ink-soft">Terakhir dihitung {dateTime(data.questions[0].computed_at)}.</p>
				{/if}
			</section>
		</div>

		<div class="space-y-6">
			{#if isEditor}
				<section class="card p-5 sm:p-6" aria-labelledby="edit">
					<h2 id="edit" class="text-xl">Pengaturan</h2>
					<label class="mt-4 block"><span class="label">Bagi hasil (%)</span><input class="field" type="number" min="0" max="100" step="0.5" bind:value={share} /></label>
					<label class="mt-3 block"><span class="label">Status</span>
						<select class="field" bind:value={status}>
							<option value="active">Aktif</option>
							<option value="inactive">Nonaktif</option>
						</select>
					</label>
					<button class="btn btn-primary mt-4" disabled={busy} onclick={save}>Simpan</button>
					{#if saveErr}<p class="field-error mt-3" role="alert">{saveErr}</p>{/if}
					{#if saveMsg}<p class="mt-3 font-semibold text-leaf-deep" role="status">{saveMsg}</p>{/if}
					<p class="mt-3 text-xs text-ink-soft">M8 mencatat porsi bagi hasil; perhitungan pembayaran belum dilakukan.</p>
				</section>

				<section class="card p-5 sm:p-6" aria-labelledby="assign">
					<h2 id="assign" class="text-xl">Tautkan soal</h2>
					<p class="mt-1 text-sm text-ink-soft">Tempel ID soal (UUID) untuk mengatribusikannya ke kreator ini.</p>
					<form class="mt-3 space-y-3" onsubmit={assign}>
						<label class="block"><span class="label">ID soal</span><input class="field font-mono text-sm" required placeholder="00000000-0000-…" bind:value={assignID} /></label>
						<button class="btn btn-primary" disabled={!assignID.trim()}>Tautkan</button>
					</form>
					{#if assignErr}<p class="field-error mt-3" role="alert">{assignErr}</p>{/if}
					{#if assignMsg}<p class="mt-3 font-semibold text-leaf-deep" role="status">{assignMsg}</p>{/if}
				</section>
			{:else}
				<section class="card p-5 sm:p-6">
					<p class="text-sm text-ink-soft">Hanya editor atau admin yang dapat mengubah kreator.</p>
				</section>
			{/if}
		</div>
	</div>
{:else if loadErr}
	<div class="card mt-6 p-6" role="alert">
		<p class="font-semibold text-[#a8264f]">{loadErr}</p>
		<p class="mt-1 text-sm text-ink-soft">Kembali ke daftar kreator untuk memilih yang lain.</p>
	</div>
{:else}
	<div class="skeleton mt-6 h-96" aria-busy="true"></div>
{/if}
