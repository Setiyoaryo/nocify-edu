<script lang="ts">
	import { api, get, post, ApiError } from '$lib/api';
	import { session } from '$lib/session.svelte';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import MagnifyingGlass from 'phosphor-svelte/lib/MagnifyingGlass';
	import Robot from 'phosphor-svelte/lib/Robot';
	import UploadSimple from 'phosphor-svelte/lib/UploadSimple';

	let assets = $state<any[]>([]);
	let licenses = $state<any[]>([]);
	let q = $state('');
	let status = $state('');
	let tag = $state('');
	let err = $state('');
	const isReviewer = $derived(session.user?.role === 'reviewer' || session.user?.role === 'admin');
	const kinds = ['illustration', 'character', 'mascot', 'icon', 'sticker', 'reward', 'background', 'shape', 'animation', 'sound', 'diagram', 'photo', 'logo'];
	const steps = ['imported', 'license_verified', 'design_approved', 'performance_checked', 'published'];
	const label: Record<string, string> = { imported: 'Diimpor', license_verified: 'Lisensi terverifikasi', design_approved: 'Desain disetujui', performance_checked: 'Performa dicek', published: 'Tayang', rejected: 'Ditolak' };

	async function load() {
		assets = (await get(`/admin/assets?${new URLSearchParams({ q, status, tag })}`)).assets;
	}
	load();
	get('/admin/licenses').then((r) => (licenses = r.licenses));

	let form = $state({ name: '', kind: 'illustration', source: '', creator: '', license_id: null as number | null, license_url: '', downloaded_at: '', original_file: '', modified: false, modification_notes: '', ai_generated: false, usage: '', tags: '' });
	let file: FileList | undefined = $state();
	let fields = $state<Record<string, string>>({});

	async function create(e: SubmitEvent) {
		e.preventDefault();
		fields = {};
		const fd = new FormData();
		fd.set('meta', JSON.stringify({ ...form, usage: form.usage.split(',').map((s) => s.trim()).filter(Boolean), tags: form.tags.split(',').map((s) => s.trim()).filter(Boolean), downloaded_at: form.downloaded_at || undefined, license_id: form.license_id ?? undefined }));
		if (file?.[0]) fd.set('file', file[0]);
		try {
			await api('POST', '/admin/assets', fd);
			form = { ...form, name: '', original_file: '', modification_notes: '' };
			await load();
		} catch (e) {
			if (e instanceof ApiError) fields = Object.keys(e.fields).length ? e.fields : { name: e.message };
		}
	}
	async function move(a: any, to: string) {
		err = '';
		try {
			await post(`/admin/assets/${a.id}/transition`, { to });
			await load();
		} catch (e) {
			err = `${a.code}: ${e instanceof ApiError ? e.message : 'gagal'}`;
		}
	}
	const kb = (n: number | null) => (n === null ? '–' : `${Math.round(n / 1024)} KB`);
</script>

<svelte:head><title>Aset | CMS</title></svelte:head>
<h1 class="text-3xl sm:text-4xl">Pustaka aset</h1>
<p class="mt-1 max-w-3xl text-ink-soft">
	Setiap aset visual dan suara wajib tercatat sumber, kreator, dan lisensinya, lalu melewati lima tahap: diimpor, lisensi terverifikasi, desain disetujui, performa dicek, dan tayang.
	Gambar hasil AI tidak dapat disetujui untuk produksi.
</p>

<div class="card mt-6 overflow-hidden">
<div class="flex flex-wrap gap-2 border-b border-line p-4">
	<div class="relative w-full sm:w-72">
		<MagnifyingGlass size={18} class="pointer-events-none absolute top-1/2 left-3.5 -translate-y-1/2 text-ink-soft" aria-hidden="true" />
		<input class="field !min-h-11 pl-10" placeholder="Cari nama, kode, sumber, kreator" aria-label="Cari aset" bind:value={q} onkeydown={(e) => e.key === 'Enter' && load()} />
	</div>
	<select class="field !min-h-11 w-56" aria-label="Status" bind:value={status} onchange={load}>
		<option value="">Semua status</option>
		{#each Object.entries(label) as [k, v]}<option value={k}>{v}</option>{/each}
	</select>
	<input class="field !min-h-11 w-48" placeholder="Filter tag" aria-label="Filter tag" bind:value={tag} onkeydown={(e) => e.key === 'Enter' && load()} />
</div>
{#if err}<p class="field-error px-4 pt-3" role="alert">{err}</p>{/if}

<ul class="divide-y divide-line xl:hidden">
	{#each assets as a (a.id)}
		{@const next = steps[steps.indexOf(a.status) + 1]}
		<li class="flex gap-3 px-4 py-4">
			{#if a.url && a.mime_type?.startsWith('image/')}<img src={a.url} alt="" class="size-14 shrink-0 rounded-lg bg-paper-deep object-contain" />{:else}<span class="size-14 shrink-0 rounded-lg bg-paper-deep"></span>{/if}
			<div class="min-w-0 flex-1">
				<div class="flex items-start justify-between gap-2"><p class="font-bold">{a.code} <span class="font-semibold text-ink-soft">{a.name}</span></p><StatusPill status={a.status} /></div>
				<p class="mt-0.5 text-sm text-ink-soft">{a.kind}, {a.source}, {a.license_name ?? 'lisensi belum dicatat'}, {kb(a.size_bytes)}</p>
				{#if isReviewer}
					<div class="mt-2 flex flex-wrap gap-2">
						{#if next}<button class="btn btn-quiet btn-sm" onclick={() => move(a, next)}>Tandai: {label[next]}</button>{/if}
						{#if a.status !== 'rejected'}<button class="btn btn-sm text-error" onclick={() => move(a, 'rejected')}>Tolak</button>{/if}
					</div>
				{/if}
			</div>
		</li>
	{:else}
		<li class="px-4 py-8 text-ink-soft">Belum ada aset.</li>
	{/each}
</ul>
<div class="hidden overflow-x-auto xl:block">
	<table class="w-full min-w-4xl text-left text-sm">
		<thead class="bg-paper-deep/60 text-xs font-bold text-ink-soft"><tr><th class="px-4 py-3">Pratinjau</th><th>Aset</th><th>Sumber & lisensi</th><th>Ukuran</th><th>Dipakai di</th><th>Status</th><th></th></tr></thead>
		<tbody>
			{#each assets as a (a.id)}
				{@const next = steps[steps.indexOf(a.status) + 1]}
				<tr class="border-t border-line align-top [&>td]:px-4">
					<td class="px-4 py-3">{#if a.url && a.mime_type?.startsWith('image/')}<img src={a.url} alt="" class="size-14 rounded-lg bg-paper-deep object-contain" />{:else}<span class="text-ink-soft">–</span>{/if}</td>
					<td><b>{a.code}</b><br />{a.name}<br /><span class="text-ink-soft">{a.kind}</span>{#if a.ai_generated}<br /><span class="pill mt-1 bg-rose-tint text-[#a8264f]"><Robot size={14} weight="fill" aria-hidden="true" />Hasil AI</span>{/if}</td>
					<td>{a.source}, oleh {a.creator}<br />{#if a.license_name}<a class="underline" href={a.license_url} rel="noopener noreferrer" target="_blank">{a.license_name}</a>{:else}<span class="text-tangerine-deep">Lisensi belum dicatat</span>{/if}{#if a.modified}<br /><span class="text-ink-soft">Dimodifikasi: {a.modification_notes}</span>{/if}</td>
					<td class={a.size_bytes > a.budget_bytes ? 'font-bold text-error' : ''}>{kb(a.size_bytes)}<br /><span class="font-normal text-ink-soft">batas {kb(a.budget_bytes)}</span></td>
					<td>{a.usage.join(', ') || '–'}</td>
					<td><StatusPill status={a.status} /></td>
					<td class="space-y-1 text-right">
						{#if isReviewer && next}<button class="btn btn-quiet btn-sm whitespace-nowrap" onclick={() => move(a, next)}>Tandai: {label[next]}</button>{/if}
						{#if isReviewer && a.status !== 'rejected'}<button class="block w-full min-h-11 text-xs text-error underline" onclick={() => move(a, 'rejected')}>Tolak</button>{/if}
					</td>
				</tr>
			{:else}
				<tr><td colspan="7" class="px-4 py-8 text-ink-soft">Belum ada aset. Belum ada ilustrasi yang lolos review brand; aset baru harus tercatat di sini sebelum dipakai.</td></tr>
			{/each}
		</tbody>
	</table>
</div>

</div>

<form class="card mt-8 grid gap-4 p-5 sm:p-6 md:grid-cols-3" onsubmit={create}>
	<h2 class="flex items-center gap-2 text-xl md:col-span-3"><UploadSimple size={24} weight="bold" class="text-primary" aria-hidden="true" />Impor aset</h2>
	<label><span class="label">Nama file</span><input class="field" bind:value={form.name} />{#if fields.name}<span class="field-error">{fields.name}</span>{/if}</label>
	<label><span class="label">Jenis</span><select class="field" bind:value={form.kind}>{#each kinds as k}<option>{k}</option>{/each}</select></label>
	<label><span class="label">File (maks. 5 MB)</span><input class="field" type="file" accept=".svg,.png,.webp,.avif,.jpg,.json,.mp3,.ogg,.webm" bind:files={file} />{#if fields.file}<span class="field-error">{fields.file}</span>{/if}</label>
	<label><span class="label">Sumber</span><input class="field" placeholder="DrawKit" bind:value={form.source} />{#if fields.source}<span class="field-error">{fields.source}</span>{/if}</label>
	<label><span class="label">Kreator</span><input class="field" bind:value={form.creator} />{#if fields.creator}<span class="field-error">{fields.creator}</span>{/if}</label>
	<label><span class="label">Lisensi</span><select class="field" bind:value={form.license_id} onchange={() => (form.license_url = licenses.find((l) => l.id === form.license_id)?.url ?? '')}>
		<option value={null}>Pilih lisensi</option>{#each licenses as l}<option value={l.id}>{l.name}</option>{/each}</select></label>
	<label><span class="label">URL lisensi</span><input class="field" bind:value={form.license_url} /></label>
	<label><span class="label">Tanggal unduh</span><input class="field" type="date" bind:value={form.downloaded_at} /></label>
	<label><span class="label">File asli</span><input class="field" placeholder="assets/illustrations/original/…" bind:value={form.original_file} /></label>
	<label class="flex min-h-11 items-center gap-3"><input type="checkbox" class="size-5 accent-primary" bind:checked={form.modified} />Dimodifikasi</label>
	<label class="md:col-span-2"><span class="label">Catatan modifikasi</span><input class="field" bind:value={form.modification_notes} /></label>
	<label class="md:col-span-2"><span class="label">Dipakai di (pisahkan koma)</span><input class="field" placeholder="landing-page, onboarding" bind:value={form.usage} /></label>
	<label><span class="label">Tag (pisahkan koma)</span><input class="field" placeholder="buah, hitung, tk-a" bind:value={form.tags} /></label>
	<label class="flex min-h-11 items-center gap-3"><input type="checkbox" class="size-5 accent-primary" bind:checked={form.ai_generated} />Dibuat dengan AI</label>
	<button class="btn btn-primary md:col-span-3 md:justify-self-start">Impor</button>
</form>
