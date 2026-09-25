<script lang="ts">
	import { get, post, patch, ApiError } from '$lib/api';
	import { session } from '$lib/session.svelte';

	let tree = $state<any>(null);
	let gradeId = $state<number | null>(null);
	let err = $state('');
	const canEdit = $derived(session.user?.role === 'content_editor' || session.user?.role === 'admin');
	async function load() {
		tree = await get('/admin/curriculum');
		gradeId ??= tree.grades[0]?.id ?? null;
	}
	load();

	const grade = $derived(tree?.grades.find((g: any) => g.id === gradeId));
	const chapters = $derived(tree ? tree.chapters.filter((c: any) => c.grade_id === gradeId) : []);

	let newChapter = $state({ subject_id: null as number | null, code: '', name: '' });
	let newSkill = $state<Record<number, { code: string; name: string }>>({});

	async function run(fn: () => Promise<unknown>) {
		err = '';
		try {
			await fn();
			await load();
		} catch (e) {
			err = e instanceof ApiError ? Object.values(e.fields)[0] ?? e.message : 'Gagal.';
		}
	}
</script>

<svelte:head><title>Kurikulum | CMS</title></svelte:head>
<h1 class="text-3xl sm:text-4xl">Kurikulum</h1>
<p class="mt-1 text-ink-soft">Susunannya: jenjang, kelas, pelajaran, bab, lalu skill. Semua disimpan sebagai data, tidak ada yang di-hardcode.</p>
{#if err}<p class="field-error mt-4" role="alert">{err}</p>{/if}

{#if tree}
	<div class="mt-6 flex flex-wrap gap-1" role="group" aria-label="Kelas">
		{#each tree.grades as g}
			<button class="min-h-10 rounded-full px-4 text-sm font-bold transition-colors {gradeId === g.id ? 'bg-primary text-white' : 'bg-paper text-ink-soft shadow-[var(--shadow-card)] hover:text-ink'}" aria-pressed={gradeId === g.id} onclick={() => (gradeId = g.id)}>{g.name}</button>
		{/each}
	</div>

	{#if grade}
		<form class="card mt-6 flex flex-wrap items-end gap-3 p-5" onsubmit={(e) => { e.preventDefault(); run(() => patch(`/admin/grades/${grade.id}`, { session_size: Number(grade.session_size), ux_band: grade.ux_band })); }}>
			<label><span class="label">Soal per sesi</span><input class="field w-28" type="number" min="3" max="30" bind:value={grade.session_size} disabled={!canEdit} /></label>
			<label><span class="label">Pola UX</span>
				<select class="field" bind:value={grade.ux_band} disabled={!canEdit}>
					<option value="early">TK (early)</option><option value="lower">SD rendah (lower)</option><option value="upper">SD tinggi (upper)</option><option value="secondary">SMP/SMA</option>
				</select></label>
			{#if canEdit}<button class="btn btn-quiet">Simpan kelas</button>{/if}
		</form>

		<div class="mt-8 space-y-8">
			{#each tree.subjects as s}
				{@const chs = chapters.filter((c: any) => c.subject_id === s.id)}
				{#if chs.length}
					<section>
						<h2 class="text-2xl">{s.name}</h2>
						{#each chs as c}
							<div class="card mt-4 p-5">
								<h3 class="text-lg">{c.name} <span class="font-sans text-sm font-normal text-ink-soft">{c.code}</span></h3>
								<table class="mt-2 w-full text-sm">
									<tbody>
										{#each tree.skills.filter((k: any) => k.chapter_id === c.id) as k}
											<tr class="border-t border-line"><td class="py-2">{k.name}</td><td class="text-ink-soft">{k.code}</td><td class="text-right">{k.live} tayang / {k.questions} soal</td></tr>
										{/each}
									</tbody>
								</table>
								{#if canEdit}
									{@const ns = (newSkill[c.id] ??= { code: '', name: '' })}
									<form class="mt-3 flex flex-wrap gap-2" onsubmit={(e) => { e.preventDefault(); run(() => post('/admin/skills', { chapter_id: c.id, ...ns })); }}>
										<input class="field !min-h-11 w-44" placeholder="kode-skill" aria-label="Kode skill" bind:value={ns.code} />
										<input class="field !min-h-11 flex-1" placeholder="Nama skill baru" aria-label="Nama skill" bind:value={ns.name} />
										<button class="btn btn-quiet !min-h-11">Tambah skill</button>
									</form>
								{/if}
							</div>
						{/each}
					</section>
				{/if}
			{/each}
		</div>

		{#if canEdit}
			<form class="card mt-10 grid gap-3 p-5 sm:grid-cols-4 sm:items-end" onsubmit={(e) => { e.preventDefault(); run(() => post('/admin/chapters', { grade_id: gradeId, ...newChapter })); }}>
				<h2 class="text-lg sm:col-span-4">Bab baru untuk {grade.name}</h2>
				<label><span class="label">Pelajaran</span><select class="field" bind:value={newChapter.subject_id}>{#each tree.subjects as s}<option value={s.id}>{s.name}</option>{/each}</select></label>
				<label><span class="label">Kode</span><input class="field" bind:value={newChapter.code} /></label>
				<label><span class="label">Nama</span><input class="field" bind:value={newChapter.name} /></label>
				<button class="btn btn-primary">Tambah bab</button>
			</form>
		{/if}
	{/if}
{/if}
