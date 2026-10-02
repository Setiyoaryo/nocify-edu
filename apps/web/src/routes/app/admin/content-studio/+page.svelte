<script lang="ts">
	// Content Studio — browse pipeline content per lesson (skill).
	// grade → subject → chapter → skill, with candidate + published counts.
	import { get, ApiError } from '$lib/api';
	import { skillHasWork, type StudioTree, type StudioGrade, type StudioSkill } from '$lib/contentStudio';
	import WarningCircle from 'phosphor-svelte/lib/WarningCircle';

	let tree = $state<StudioTree | null>(null);
	let err = $state('');
	let gradeId = $state<number | null>(null);

	get('/admin/content-studio/tree')
		.then((r: StudioTree) => {
			tree = r;
			gradeId ??= r.grades[0]?.id ?? null;
		})
		.catch((e) => (err = e instanceof ApiError ? e.message : 'Gagal memuat Content Studio.'));

	const grade = $derived(tree?.grades.find((g) => g.id === gradeId) ?? null);
	const total = $derived(
		tree
			? tree.grades.reduce(
					(n, g) =>
						n +
						g.subjects.reduce(
							(m, s) =>
								m +
								s.chapters.reduce(
									(k, c) => k + c.skills.reduce((j, sk) => j + sk.materials + sk.questions, 0),
									0
								),
							0
						),
					0
				)
			: 0
	);

	function skillsWithWork(g: StudioGrade): { subject: string; chapter: string; skills: StudioSkill[] }[] {
		const out: { subject: string; chapter: string; skills: StudioSkill[] }[] = [];
		for (const s of g.subjects) {
			for (const c of s.chapters) {
				const skills = c.skills.filter(skillHasWork);
				if (skills.length) out.push({ subject: s.name, chapter: c.name, skills });
			}
		}
		return out;
	}
</script>

<svelte:head><title>Content Studio | CMS EduPlay</title></svelte:head>
<div class="flex flex-wrap items-center justify-between gap-4">
	<div>
		<h1 class="text-3xl sm:text-4xl">Content Studio</h1>
		<p class="mt-1 text-ink-soft">Telusuri konten pipeline per pelajaran: materi, soal turunannya, dan soal yang sudah tayang.</p>
	</div>
</div>

{#if err}
	<p class="field-error mt-6" role="alert">{err}</p>
{:else if tree && !tree.available}
	<div class="card mt-6 p-6">
		<p class="text-ink-soft">Pipeline konten belum dijalankan di lingkungan ini.</p>
		<p class="mt-1 text-sm text-ink-soft">Jalankan tahap fetching dan ekstraksi lebih dulu, lalu muat ulang halaman ini.</p>
	</div>
{:else if tree}
	{#if tree.unmapped.materials || tree.unmapped.questions}
		<div class="card mt-6 flex flex-wrap items-center gap-3 border-l-4 border-l-sun p-4">
			<WarningCircle size={20} weight="fill" class="text-sun-deep" aria-hidden="true" />
			<p class="text-sm">
				<b>Belum dipetakan:</b> {tree.unmapped.materials} materi dan {tree.unmapped.questions} soal belum punya skill,
				jadi tidak muncul di bawah. Tinjau di <a class="font-bold text-primary-deep hover:underline" href="/app/admin/review">antrian review</a>.
			</p>
		</div>
	{/if}

	<div class="mt-6 flex flex-wrap gap-1" role="group" aria-label="Kelas">
		{#each tree.grades as g}
			<button class="min-h-10 rounded-full px-4 text-sm font-bold transition-colors {gradeId === g.id ? 'bg-primary text-white' : 'bg-paper text-ink-soft shadow-[var(--shadow-card)] hover:text-ink'}" aria-pressed={gradeId === g.id} onclick={() => (gradeId = g.id)}>{g.name}</button>
		{/each}
	</div>

	{#if total === 0}
		<div class="card mt-6 p-6 text-center">
			<p class="text-ink-soft">Belum ada materi atau soal pada pipeline. Jalankan pipeline konten lebih dulu.</p>
		</div>
	{:else if grade}
		{@const groups = skillsWithWork(grade)}
		{#if groups.length === 0}
			<div class="card mt-6 p-6 text-center">
				<p class="text-ink-soft">Tidak ada konten untuk kelas ini.</p>
			</div>
		{:else}
			<div class="mt-8 space-y-8">
				{#each groups as grp}
					<section>
						<p class="text-xs font-bold text-ink-soft uppercase">{grp.subject}</p>
						<div class="card mt-2 overflow-hidden">
							<h2 class="border-b border-line px-4 py-3 text-lg">{grp.chapter}</h2>
							<table class="w-full text-left text-sm">
								<thead class="bg-paper-deep/60 text-xs font-bold text-ink-soft">
									<tr>
										<th class="px-4 py-2">Pelajaran (skill)</th>
										<th class="px-4 py-2 text-right">Materi</th>
										<th class="px-4 py-2 text-right">Soal</th>
										<th class="px-4 py-2 text-right">Perlu tinjauan</th>
										<th class="px-4 py-2 text-right">Tayang</th>
									</tr>
								</thead>
								<tbody>
									{#each grp.skills as sk}
										<tr class="border-t border-line hover:bg-paper-deep/40">
											<td class="px-4 py-3">
												<a class="font-bold text-ink hover:text-primary-deep" href="/app/admin/content-studio/{sk.id}">{sk.name || sk.code}</a>
												<span class="ml-2 font-mono text-xs text-ink-soft">{sk.code}</span>
											</td>
											<td class="px-4 text-right">{sk.materials}</td>
											<td class="px-4 text-right">{sk.questions}</td>
											<td class="px-4 text-right">
												{#if sk.needs_review}
													<span class="pill bg-sun-tint text-ink">{sk.needs_review}</span>
												{:else}
													<span class="text-ink-soft">0</span>
												{/if}
											</td>
											<td class="px-4 text-right">{sk.published}</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</section>
				{/each}
			</div>
		{/if}
	{/if}
{:else}
	<div class="skeleton mt-6 h-64" aria-busy="true"></div>
{/if}
