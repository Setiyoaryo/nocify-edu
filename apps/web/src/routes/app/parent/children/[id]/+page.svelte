<script lang="ts">
	import { get, patch, del, post, track, ApiError } from '$lib/api';
	import { starterAvatars } from '$lib/avatars';
	import { date, dateTime, relative } from '$lib/format';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import MasteryBar from '$lib/components/MasteryBar.svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import StatTile from '$lib/components/StatTile.svelte';
	import Recommendation from '$lib/components/Recommendation.svelte';
	import ChildForm from '../ChildForm.svelte';
	import progressArt from '$lib/illustrations/progress.svg';
	import missionsArt from '$lib/illustrations/missions.svg';
	import Fire from 'phosphor-svelte/lib/Fire';
	import GameController from 'phosphor-svelte/lib/GameController';
	import Target from 'phosphor-svelte/lib/Target';
	import CheckSquare from 'phosphor-svelte/lib/CheckSquare';
	import ListChecks from 'phosphor-svelte/lib/ListChecks';
	import Crosshair from 'phosphor-svelte/lib/Crosshair';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	import CircleHalf from 'phosphor-svelte/lib/CircleHalf';
	import WarningCircle from 'phosphor-svelte/lib/WarningCircle';
	import Flag from 'phosphor-svelte/lib/Flag';
	import LockKey from 'phosphor-svelte/lib/LockKey';
	import Star from 'phosphor-svelte/lib/Star';
	import Sparkle from 'phosphor-svelte/lib/Sparkle';
	import X from 'phosphor-svelte/lib/X';
	import type { Child } from '$lib/types';

	const id = page.params.id as string;
	type Tab = 'progress' | 'missions' | 'history' | 'profile';
	const initialTab = page.url.searchParams.get('tab');
	let tab = $state<Tab>((['progress', 'missions', 'history', 'profile'] as const).find((t) => t === initialTab) ?? 'progress');
	let progress = $state<any>(null);
	let history = $state<any>(null);
	let recs = $state<any>(null);
	let missions = $state<any[]>([]);
	let skills = $state<any[]>([]);
	let unlocked = $state<string[]>(starterAvatars);
	let error = $state('');

	async function loadMissions() {
		missions = (await get(`/children/${id}/missions`)).missions;
	}
	Promise.all([get(`/children/${id}/progress`), get(`/children/${id}/history`), get(`/children/${id}/recommendations`), get(`/children/${id}/rewards`), loadMissions()])
		.then(async ([p, h, r, rw]) => {
			progress = p;
			history = h;
			recs = r;
			unlocked = rw.items.filter((i: any) => i.kind === 'avatar' && i.owned).map((i: any) => i.code);
			form = { nickname: p.child.nickname, grade_id: p.child.grade_id, avatar: p.child.avatar, pin: '' };
			skills = (await get(`/skills?grade_id=${p.child.grade_id}`)).skills;
			track('progress_viewed');
		})
		.catch((e) => (error = e.message));

	const child = $derived<Child | null>(progress?.child ?? null);
	const activeMissions = $derived(missions.filter((m) => m.status === 'active'));
	const groups = [
		{ key: 'strong', title: 'Kuat', icon: CheckCircle, cls: 'bg-leaf-tint text-leaf-deep' },
		{ key: 'developing', title: 'Berkembang', icon: CircleHalf, cls: 'bg-sun-tint text-ink' },
		{ key: 'needs_practice', title: 'Perlu latihan', icon: WarningCircle, cls: 'bg-tangerine-tint text-tangerine-deep' }
	] as const;
	const tabs: [Tab, string][] = [['progress', 'Perkembangan'], ['missions', 'Misi'], ['history', 'Riwayat'], ['profile', 'Profil']];

	// missions
	let newMission = $state({ skill_id: null as number | null, target: 5 });
	let missionErr = $state('');
	async function createMission(e: SubmitEvent) {
		e.preventDefault();
		missionErr = '';
		try {
			missions = (await post(`/children/${id}/missions`, newMission)).missions;
			newMission = { skill_id: null, target: 5 };
		} catch (err) {
			missionErr = err instanceof ApiError ? (Object.values(err.fields)[0] ?? err.message) : 'Gagal.';
		}
	}
	async function cancelMission(mid: string) {
		missions = (await del(`/children/${id}/missions/${mid}`)).missions;
	}

	// profile
	let form = $state({ nickname: '', grade_id: null as number | null, avatar: 'cat', pin: '' });
	let fields = $state<Record<string, string>>({});
	let saved = $state(false);
	async function save(e: SubmitEvent) {
		e.preventDefault();
		fields = {};
		saved = false;
		try {
			const r = await patch(`/children/${id}`, { ...form, pin: form.pin || undefined });
			progress = { ...progress, child: r.child };
			skills = (await get(`/skills?grade_id=${r.child.grade_id}`)).skills;
			saved = true;
		} catch (err) {
			if (err instanceof ApiError) fields = Object.keys(err.fields).length ? err.fields : { nickname: err.message };
		}
	}
	async function removePin() {
		const r = await patch(`/children/${id}`, { pin: '' });
		progress = { ...progress, child: r.child };
	}
	async function remove() {
		if (!child) return;
		const typed = prompt(`Hapus profil ${child.nickname} beserta seluruh riwayat belajarnya? Ini tidak bisa dibatalkan.\n\nKetik nama panggilan untuk konfirmasi:`);
		if (typed?.trim() !== child.nickname) return;
		await del(`/children/${id}`);
		goto('/app/parent/children');
	}
</script>

<svelte:head><title>{child ? `Laporan ${child.nickname}` : 'Laporan'} | EduPlay</title></svelte:head>

{#if error}
	<p class="field-error" role="alert">{error}</p>
{:else if !progress || !child}
	<div class="card skeleton h-48" aria-busy="true"></div>
{:else}
	<header class="pastel-grape anim-rise flex flex-wrap items-center gap-4 rounded-[1.6rem] p-5 sm:p-7">
		<Avatar code={child.avatar} size={68} ring />
		<div class="min-w-0 flex-1">
			<h1 class="text-3xl">{child.nickname}</h1>
			<p class="text-ink-soft">{child.grade_name}, level {child.level} ({child.xp} XP), aktif {relative(child.last_activity).toLowerCase()}</p>
		</div>
		<div class="flex w-full flex-wrap items-center gap-2 sm:w-auto">
			{#if progress.streak > 0}<span class="pill bg-white text-tangerine-deep shadow-sm"><Fire size={16} weight="fill" aria-hidden="true" />{progress.streak} hari beruntun</span>{/if}
			<a href="/app/play/{child.id}/card" class="btn border border-primary/20 bg-white text-primary hover:bg-primary-tint w-full sm:w-auto"><Sparkle size={18} weight="fill" class="text-sun-deep" aria-hidden="true" />Kartu Legendaris</a>
			<a href="/app/play/{child.id}" class="btn btn-sun w-full sm:w-auto"><GameController size={20} weight="fill" aria-hidden="true" />Main sebagai {child.nickname}</a>
		</div>
	</header>

	<div class="mt-6 grid grid-cols-[1.45fr_1fr_1fr_1fr] gap-1 rounded-full bg-paper p-1 shadow-[var(--shadow-card)] sm:inline-grid sm:grid-flow-col sm:grid-cols-none" role="tablist" aria-label="Laporan">
		{#each tabs as [k, label]}
			<button role="tab" id="tab-{k}" aria-selected={tab === k} aria-controls="panel-{k}"
				class="min-h-11 rounded-full px-1 text-[0.8rem] font-bold whitespace-nowrap transition-colors min-[400px]:text-sm sm:px-5 sm:text-base {tab === k ? 'bg-primary text-white' : 'text-ink-soft hover:text-ink'}"
				onclick={() => (tab = k)}>{label}{#if k === 'missions' && activeMissions.length}<span class="ml-1.5 rounded-full bg-sun px-1.5 text-xs text-ink">{activeMissions.length}</span>{/if}</button>
		{/each}
	</div>

	{#if tab === 'progress'}
		<div id="panel-progress" role="tabpanel" aria-labelledby="tab-progress" class="anim-rise mt-6 space-y-6">
			<div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
				<StatTile label="Challenge dimulai (30 hari)" value={progress.activity.started_30d} icon={Target} />
				<StatTile label="Diselesaikan (30 hari)" value={progress.activity.completed_30d} icon={CheckSquare} tint="bg-leaf-tint text-leaf-deep" />
				<StatTile label="Soal (7 hari)" value={progress.activity.questions_7d} icon={ListChecks} tint="bg-sky-tint text-sky" />
				<StatTile label="Akurasi (7 hari)" value={progress.activity.accuracy_7d === null ? '–' : Math.round(progress.activity.accuracy_7d) + '%'} icon={Crosshair} tint="bg-sun-tint text-ink" />
			</div>

			{#if progress.overall === null}
				<div class="card grid items-center gap-6 p-6 sm:grid-cols-[1fr_auto] sm:p-8">
					<p class="text-lg">{child.nickname} belum menyelesaikan challenge. Setelah challenge pertama, laporan per pelajaran, bab, dan skill muncul di sini.</p>
					<img src={progressArt} alt="" class="mx-auto w-48" />
				</div>
			{:else}
				<section class="card p-5 sm:p-6">
					<div class="flex items-center justify-between gap-3">
						<h2 class="text-xl">Per pelajaran dan bab</h2>
						<span class="text-sm font-semibold text-ink-soft">Keseluruhan <b class="text-ink">{Math.round(progress.overall)}%</b></span>
					</div>
					<div class="mt-5 grid gap-5 lg:grid-cols-2">
						{#each progress.subjects as s}
							<div class="rounded-2xl border border-line p-4">
								<MasteryBar label={s.name} value={s.mastery} />
								<p class="mt-1 text-xs text-ink-soft">{s.skills_practiced} dari {s.skills_total} skill sudah dicoba</p>
								<div class="mt-4 space-y-3 border-t border-line pt-4">
									{#each progress.chapters.filter((c: any) => c.subject_id === s.subject_id) as c}
										<MasteryBar label={c.name} value={c.mastery} compact />
									{/each}
								</div>
							</div>
						{/each}
					</div>
				</section>

				<section class="card p-5 sm:p-6">
					<h2 class="text-xl">Skill</h2>
					<div class="mt-5 grid gap-5 md:grid-cols-3">
						{#each groups as g}
							<div>
								<h3 class="flex items-center gap-2 text-base"><span class="grid size-8 place-items-center rounded-lg {g.cls}"><g.icon size={18} weight="fill" aria-hidden="true" /></span>{g.title}</h3>
								<ul class="mt-3 space-y-2">
									{#each progress.skills[g.key] as s}
										<li class="rounded-xl bg-paper-deep/70 p-3">
											<p class="font-bold">{s.name}</p>
											<p class="text-sm text-ink-soft">{s.subject_name}, {Math.round(s.mastery)}% dari {s.evidence} soal</p>
										</li>
									{:else}
										<li class="text-sm text-ink-soft">Belum ada.</li>
									{/each}
								</ul>
							</div>
						{/each}
					</div>
					{#if progress.insufficient > 0}<p class="mt-4 text-sm text-ink-soft">{progress.insufficient} skill lain belum cukup data (minimal 3 soal).</p>{/if}
					{#if progress.locked.skills && progress.hidden_skills > 0}
						<div class="mt-5 flex flex-col gap-3 rounded-2xl bg-night p-4 text-white sm:flex-row sm:items-center">
							<LockKey size={24} weight="duotone" class="shrink-0 text-sun" aria-hidden="true" />
							<p class="flex-1">{progress.hidden_skills} skill lain tersedia di laporan lengkap.</p>
							<a class="btn btn-sun btn-sm" href="/app/parent/subscription">Berlangganan</a>
						</div>
					{/if}
				</section>

				{#if progress.trend?.length}
					<section class="card p-5 sm:p-6">
						<h2 class="text-xl">Tren mingguan</h2>
						<div class="mt-4 overflow-x-auto">
							<table class="w-full min-w-md text-left">
								<thead class="text-sm text-ink-soft"><tr><th class="py-2 font-semibold">Minggu mulai</th><th class="font-semibold">Soal</th><th class="font-semibold">Akurasi jawaban pertama</th></tr></thead>
								<tbody>
									{#each progress.trend as t}
										<tr class="border-t border-line"><td class="py-2.5">{date(t.day)}</td><td class="tabular-nums">{t.questions}</td><td class="font-bold tabular-nums">{Math.round(t.accuracy)}%</td></tr>
									{/each}
								</tbody>
							</table>
						</div>
					</section>
				{/if}
			{/if}

			<section class="card p-5 sm:p-6">
				<h2 class="text-xl">Saran berikutnya</h2>
				<div class="mt-4 space-y-3">
					{#each recs.recommendations as r}
						<Recommendation rec={r} childId={id} missionSkills={activeMissions.map((m) => m.skill_id)} onmission={loadMissions} />
					{:else}
						<p class="text-ink-soft">Belum ada saran. Selesaikan satu challenge dulu.</p>
					{/each}
				</div>
				{#if recs.locked}<p class="mt-3 text-sm">{recs.total - recs.recommendations.length} saran lain tersedia untuk pelanggan. <a class="font-bold text-primary-deep hover:underline" href="/app/parent/subscription">Lihat paket</a></p>{/if}
			</section>
		</div>
	{:else if tab === 'missions'}
		<div id="panel-missions" role="tabpanel" aria-labelledby="tab-missions" class="anim-rise mt-6 grid gap-6 lg:grid-cols-[1.3fr_1fr]">
			<section class="card p-5 sm:p-6">
				<h2 class="text-xl">Misi {child.nickname}</h2>
				<p class="mt-1 text-sm text-ink-soft">Misi muncul paling atas di beranda anak. Soalnya fokus pada skill itu. Maksimal 3 misi aktif.</p>
				<ul class="mt-5 space-y-3">
					{#each missions as m (m.id)}
						<li class="anim-rise flex items-center gap-3 rounded-2xl border border-line p-4">
							<span class="grid size-11 shrink-0 place-items-center rounded-xl {m.status === 'completed' ? 'bg-leaf-tint text-leaf-deep' : 'bg-tangerine-tint text-tangerine-deep'}">
								{#if m.status === 'completed'}<CheckCircle size={24} weight="fill" aria-hidden="true" />{:else}<Flag size={24} weight="fill" aria-hidden="true" />{/if}
							</span>
							<div class="min-w-0 flex-1">
								<p class="font-bold">{m.skill_name}</p>
								<p class="text-sm text-ink-soft">{m.subject_name}, {m.status === 'completed' ? `selesai ${relative(m.completed_at).toLowerCase()}` : m.due_at ? `sampai ${date(m.due_at)}` : ''}</p>
								<div class="mt-2 h-2 overflow-hidden rounded-full bg-paper-deep" role="img" aria-label="{m.completed} dari {m.target} challenge">
									<div class="bar-fill h-full rounded-full {m.status === 'completed' ? 'bg-leaf' : 'bg-tangerine'}" style="width:{Math.min(100, (m.completed / m.target) * 100)}%"></div>
								</div>
							</div>
							<span class="shrink-0 font-extrabold tabular-nums">{m.completed}/{m.target}</span>
							{#if m.status === 'active'}
								<button class="grid size-10 shrink-0 place-items-center rounded-full text-ink-soft hover:bg-paper-deep hover:text-error" onclick={() => cancelMission(m.id)} aria-label="Batalkan misi {m.skill_name}"><X size={18} weight="bold" aria-hidden="true" /></button>
							{/if}
						</li>
					{:else}
						<li class="flex flex-col items-center gap-3 rounded-2xl bg-paper-deep/70 p-6 text-center text-ink-soft">
							<img src={missionsArt} alt="" class="w-40" />
							Belum ada misi. Pilih skill di samping, atau jadikan saran di tab Perkembangan sebagai misi.
						</li>
					{/each}
				</ul>
			</section>
			<form class="card self-start p-5 sm:p-6" onsubmit={createMission}>
				<h2 class="text-xl">Buat misi baru</h2>
				<label class="mt-4 block"><span class="label">Skill</span>
					<select class="field" bind:value={newMission.skill_id} required>
						<option value={null} disabled>Pilih skill</option>
						{#each skills as s}<option value={s.id}>{s.chapter_name}: {s.name}</option>{/each}
					</select></label>
				<label class="mt-4 block"><span class="label">Jumlah challenge</span>
					<input class="field max-w-32" type="number" min="1" max="20" bind:value={newMission.target} /></label>
				<p class="mt-2 text-sm text-ink-soft">Berlaku 7 hari. {child.nickname} mendapat lencana "Misi Tuntas" saat selesai.</p>
				{#if missionErr}<p class="field-error" role="alert">{missionErr}</p>{/if}
				<button class="btn btn-primary mt-5 w-full" disabled={!newMission.skill_id}><Flag size={18} weight="fill" aria-hidden="true" />Buat misi</button>
			</form>
		</div>
	{:else if tab === 'history'}
		<div id="panel-history" role="tabpanel" aria-labelledby="tab-history" class="card anim-rise mt-6 p-5 sm:p-6">
			{#if history.history.length === 0}
				<p class="text-ink-soft">Belum ada challenge yang diselesaikan.</p>
			{:else}
				<div class="overflow-x-auto">
					<table class="w-full min-w-lg text-left">
						<thead class="text-sm text-ink-soft"><tr><th class="py-2 font-semibold">Waktu</th><th class="font-semibold">Pelajaran</th><th class="font-semibold">Skor</th><th class="font-semibold">Bintang</th><th class="font-semibold">Akurasi</th><th class="font-semibold">Soal</th></tr></thead>
						<tbody>
							{#each history.history as h}
								<tr class="border-t border-line">
									<td class="py-3">{dateTime(h.completed_at ?? h.started_at)}</td>
									<td class="font-semibold">{h.subject}</td>
									<td class="text-lg font-extrabold tabular-nums">{h.score}</td>
									<td><span class="flex text-sun-deep" aria-label="{h.stars} bintang">{#each [1, 2, 3] as n}<Star size={18} weight="fill" class={n <= h.stars ? '' : 'text-line'} aria-hidden="true" />{/each}</span></td>
									<td class="tabular-nums">{Math.round(h.accuracy)}%</td>
									<td class="tabular-nums">{h.answered}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
				{#if history.locked}<p class="mt-4 text-sm">Menampilkan {history.history.length} dari {history.total}. <a class="font-bold text-primary-deep hover:underline" href="/app/parent/subscription">Berlangganan</a> untuk riwayat lengkap.</p>{/if}
			{/if}
		</div>
	{:else}
		<div id="panel-profile" role="tabpanel" aria-labelledby="tab-profile" class="anim-rise mt-6 grid gap-6 lg:grid-cols-[1.3fr_1fr]">
			<form class="card p-5 sm:p-8" onsubmit={save} novalidate>
				<ChildForm bind:value={form} {fields} unlockedAvatars={unlocked} editing />
				<div class="mt-8 flex flex-wrap items-center gap-3">
					<button class="btn btn-primary">Simpan perubahan</button>
					{#if child.has_pin}<button type="button" class="btn btn-quiet" onclick={removePin}>Hapus PIN</button>{/if}
					{#if saved}<p class="flex items-center gap-1.5 font-bold text-leaf-deep" role="status"><CheckCircle size={18} weight="fill" aria-hidden="true" />Tersimpan</p>{/if}
				</div>
			</form>
			<section class="card self-start p-5 sm:p-6">
				<h2 class="text-lg">Hapus profil</h2>
				<p class="mt-1 text-sm text-ink-soft">Menghapus profil beserta seluruh jawaban, progres, misi, dan hadiah {child.nickname}. Tidak bisa dibatalkan.</p>
				<button class="btn btn-danger btn-sm mt-4" onclick={remove}>Hapus profil {child.nickname}</button>
			</section>
		</div>
	{/if}
{/if}
