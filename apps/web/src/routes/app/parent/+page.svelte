<script lang="ts">
	import { get } from '$lib/api';
	import { relative } from '$lib/format';
	import { session } from '$lib/session.svelte';
	import MasteryBar from '$lib/components/MasteryBar.svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import StatTile from '$lib/components/StatTile.svelte';
	import Recommendation from '$lib/components/Recommendation.svelte';
	import friends from '$lib/illustrations/friends.svg';
	import Fire from 'phosphor-svelte/lib/Fire';
	import Target from 'phosphor-svelte/lib/Target';
	import ListChecks from 'phosphor-svelte/lib/ListChecks';
	import Crosshair from 'phosphor-svelte/lib/Crosshair';
	import Flag from 'phosphor-svelte/lib/Flag';
	import Crown from 'phosphor-svelte/lib/Crown';
	import Plus from 'phosphor-svelte/lib/Plus';
	import FunCard from '$lib/components/FunCard.svelte';
	import GameController from 'phosphor-svelte/lib/GameController';
	import ChartPieSlice from 'phosphor-svelte/lib/ChartPieSlice';
	import UserPlus from 'phosphor-svelte/lib/UserPlus';
	import type { Child, Entitlement } from '$lib/types';

	type Overview = { child: Child; progress: any; rec: any; missions: any[] };
	let data = $state<{ items: Overview[]; sub: Entitlement } | null>(null);
	let error = $state('');

	async function loadChild(child: Child): Promise<Overview> {
		const [progress, rec, m] = await Promise.all([
			get(`/children/${child.id}/progress`),
			get(`/children/${child.id}/recommendations`),
			get(`/children/${child.id}/missions`)
		]);
		return { child, progress, rec, missions: m.missions.filter((x: any) => x.status === 'active') };
	}
	async function load() {
		try {
			const [{ children }, { subscription }] = await Promise.all([get('/children'), get('/subscription')]);
			data = { items: await Promise.all(children.map(loadChild)), sub: subscription };
		} catch (e: any) {
			error = e.message;
		}
	}
	async function refresh(i: number) {
		if (data) data.items[i] = await loadChild(data.items[i].child);
	}
	load();
	const hour = new Date().getHours();
	const greeting = hour < 11 ? 'Selamat pagi' : hour < 15 ? 'Selamat siang' : hour < 18 ? 'Selamat sore' : 'Selamat malam';
</script>

<svelte:head><title>Ringkasan | EduPlay</title></svelte:head>

<section class="pastel-lilac anim-rise relative overflow-hidden rounded-[1.6rem] p-6 sm:p-8">
	<img src={friends} alt="" class="pointer-events-none absolute -right-2 -bottom-4 hidden w-56 md:block" />
	<p class="relative font-bold text-ink-soft">{greeting},</p>
	<h1 class="relative text-3xl sm:text-4xl">{session.user?.display_name.split(' ')[0]}</h1>
	<p class="relative mt-2 max-w-lg font-semibold text-ink-soft">Pantau apa yang sudah dikuasai anak, lalu tentukan latihan berikutnya.</p>
</section>

{#if data?.items.length}
	{@const first = data.items[0].child}
	<div class="mt-6 grid grid-cols-2 gap-4 lg:grid-cols-4">
		<div class="anim-rise" style="--d:60ms"><FunCard tone="sun" icon={GameController} iconColor="#c9501c" title="Mode anak" desc="Serahkan ke anak untuk bermain" href="/app/play" /></div>
		<div class="anim-rise" style="--d:120ms"><FunCard tone="sky" icon={ChartPieSlice} iconColor="#3a8ee8" title="Laporan {first.nickname}" desc="Skill kuat & perlu latihan" href="/app/parent/children/{first.id}" /></div>
		<div class="anim-rise" style="--d:180ms"><FunCard tone="peach" icon={Flag} iconColor="#c9501c" title="Buat misi" desc="Tentukan latihan minggu ini" href="/app/parent/children/{first.id}?tab=missions" /></div>
		<div class="anim-rise" style="--d:240ms"><FunCard tone="mint" icon={UserPlus} iconColor="#15804a" title="Tambah anak" desc="Satu profil untuk tiap anak" href="/app/parent/children/new" /></div>
	</div>
{/if}

{#if error}
	<p class="field-error mt-6" role="alert">{error}</p>
{:else if !data}
	<div class="mt-8 grid gap-5" aria-busy="true" aria-live="polite">
		<div class="card skeleton h-64"></div>
	</div>
{:else if data.items.length === 0}
	<div class="card anim-rise mt-8 grid items-center gap-6 overflow-hidden p-6 sm:p-10 md:grid-cols-[1fr_auto]">
		<div>
			<h2 class="text-2xl sm:text-3xl">Buat profil anak untuk memulai</h2>
			<p class="mt-2 max-w-md text-ink-soft">Cukup nama panggilan dan kelas. Setelah itu anak bisa langsung mencoba challenge pertamanya.</p>
			<a href="/app/parent/children/new" class="btn btn-primary mt-6">Buat profil anak</a>
		</div>
		<img src={friends} alt="" class="anim-float mx-auto w-56" />
	</div>
{:else}
	{#if !data.sub.premium}
		<aside class="anim-rise mt-6 flex flex-col gap-4 rounded-[1.25rem] bg-night p-5 text-white sm:flex-row sm:items-center sm:p-6">
			<span class="grid size-12 shrink-0 place-items-center rounded-2xl bg-sun text-ink"><Crown size={26} weight="fill" aria-hidden="true" /></span>
			<div class="flex-1">
				<p class="text-lg font-extrabold">Buka laporan per skill lengkap</p>
				<p class="text-sm text-white/75">Paket gratis menampilkan ringkasan. Berlangganan untuk semua skill, riwayat, tren, dan challenge tanpa batas.</p>
			</div>
			<a href="/app/parent/subscription" class="btn btn-sun shrink-0">Lihat paket</a>
		</aside>
	{/if}

	<div class="mt-6 space-y-6">
		{#each data.items as { child, progress, rec, missions }, i (child.id)}
			<section class="card anim-rise overflow-hidden" style="--d:{i * 110}ms" aria-labelledby="c-{child.id}">
				<div class="pastel-grape flex flex-wrap items-center gap-4 p-5 sm:p-6">
					<Avatar code={child.avatar} size={56} ring />
					<div class="min-w-0 flex-1">
						<h2 id="c-{child.id}" class="text-2xl">{child.nickname}</h2>
						<p class="text-sm text-ink-soft">{child.grade_name}, level {child.level}, aktif {relative(child.last_activity).toLowerCase()}</p>
					</div>
					<div class="flex w-full flex-wrap items-center gap-2 sm:w-auto">
						{#if progress.streak > 0}
							<span class="pill bg-white text-tangerine-deep shadow-sm"><Fire size={16} weight="fill" aria-hidden="true" />{progress.streak} hari beruntun</span>
						{/if}
						<a href="/app/parent/children/{child.id}" class="pill-action ml-auto">Lihat laporan</a>
					</div>
				</div>

				<div class="grid gap-6 p-5 sm:p-6 lg:grid-cols-[1.1fr_1fr]">
					<div>
						{#if progress.overall === null}
							<p class="rounded-2xl bg-paper-deep p-5 text-ink-soft">{child.nickname} belum menyelesaikan challenge. Laporan muncul setelah challenge pertama.</p>
						{:else}
							<div class="space-y-4">
								<MasteryBar label="Keseluruhan" value={progress.overall} />
								{#each progress.subjects as s}<MasteryBar label={s.name} value={s.mastery} />{/each}
							</div>
						{/if}
						<div class="mt-6 grid grid-cols-3 gap-2 sm:gap-3">
							<StatTile label="Challenge (7 hari)" value={progress.activity.challenges_7d} icon={Target} />
							<StatTile label="Soal (7 hari)" value={progress.activity.questions_7d} icon={ListChecks} tint="bg-sky-tint text-sky" />
							<StatTile label="Akurasi (7 hari)" value={progress.activity.accuracy_7d === null ? '–' : Math.round(progress.activity.accuracy_7d) + '%'} icon={Crosshair} tint="bg-leaf-tint text-leaf-deep" />
						</div>
					</div>
					<div class="space-y-3">
						<p class="text-sm font-bold text-ink-soft">Yang perlu dilakukan</p>
						{#each missions as m (m.id)}
							<div class="flex items-center gap-3 rounded-2xl border border-line p-4">
								<span class="grid size-10 shrink-0 place-items-center rounded-xl bg-tangerine-tint text-tangerine-deep"><Flag size={22} weight="fill" aria-hidden="true" /></span>
								<div class="min-w-0 flex-1">
									<p class="truncate font-bold">Misi: {m.skill_name}</p>
									<div class="mt-1.5 h-2 overflow-hidden rounded-full bg-paper-deep" role="img" aria-label="{m.completed} dari {m.target} challenge">
										<div class="bar-fill h-full rounded-full bg-tangerine" style="width:{(m.completed / m.target) * 100}%"></div>
									</div>
								</div>
								<span class="shrink-0 text-sm font-extrabold tabular-nums">{m.completed}/{m.target}</span>
							</div>
						{/each}
						{#each rec.recommendations.slice(0, missions.length ? 1 : 2) as r}
							<Recommendation rec={r} childId={child.id} missionSkills={missions.map((m: any) => m.skill_id)} onmission={() => refresh(i)} />
						{:else}
							{#if !missions.length}<p class="rounded-2xl bg-paper-deep p-4 text-sm text-ink-soft">Saran akan muncul setelah {child.nickname} menyelesaikan challenge.</p>{/if}
						{/each}
					</div>
				</div>
			</section>
		{/each}
	</div>
{/if}
