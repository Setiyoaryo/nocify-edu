<script lang="ts">
	import { get, post, ApiError } from '$lib/api';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import FunCard from '$lib/components/FunCard.svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import play from '$lib/illustrations/play.svg';
	import MathOperations from 'phosphor-svelte/lib/MathOperations';
	import BookOpenText from 'phosphor-svelte/lib/BookOpenText';
	import Plant from 'phosphor-svelte/lib/Plant';
	import Star from 'phosphor-svelte/lib/Star';
	import Flag from 'phosphor-svelte/lib/Flag';
	import Fire from 'phosphor-svelte/lib/Fire';
	import Gift from 'phosphor-svelte/lib/Gift';
	import Lightning from 'phosphor-svelte/lib/Lightning';
	import Sparkle from 'phosphor-svelte/lib/Sparkle';
	import type { Component } from 'svelte';

	const id = page.params.childId as string;
	// One pastel family; each subject keeps its own colour (§26.3).
	const looks: Record<string, { icon: Component<any>; tone: 'grape' | 'peach' | 'mint'; color: string }> = {
		number: { icon: MathOperations, tone: 'grape', color: '#5b5ce2' },
		book: { icon: BookOpenText, tone: 'peach', color: '#c9501c' },
		nature: { icon: Plant, tone: 'mint', color: '#15804a' }
	};
	let home = $state<any>(null);
	let starting = $state<string | null>(null);
	let notice = $state('');

	get(`/children/${id}/home`).then((h) => (home = h));

	async function start(key: string, body: Record<string, unknown>) {
		starting = key;
		notice = '';
		try {
			const { assessment } = await post('/assessments', { child_id: id, ...body });
			await goto(`/app/play/${id}/challenge/${assessment.id}`);
		} catch (e) {
			notice = e instanceof ApiError ? e.message : 'Belum bisa mulai. Coba lagi.';
			starting = null;
		}
	}
	const outOfChallenges = $derived(home?.challenges_left_today === 0);
	const hour = new Date().getHours();
	const greet = hour < 11 ? 'Selamat pagi' : hour < 15 ? 'Selamat siang' : hour < 18 ? 'Selamat sore' : 'Selamat malam';
</script>

<svelte:head><title>Main | EduPlay</title></svelte:head>

{#if home}
	<!-- Greeting -->
	<section class="pastel-lilac anim-rise relative mt-3 overflow-hidden rounded-[1.6rem] p-5 sm:p-7">
		<img src={play} alt="" class="pointer-events-none absolute -right-4 -bottom-3 hidden w-52 opacity-90 sm:block" />
		<div class="relative flex items-center gap-4">
			<Avatar code={home.child.avatar} size={64} ring />
			<div>
				<p class="text-sm font-bold text-ink-soft">{greet},</p>
				<h1 class="text-3xl">Halo, {home.child.nickname}!</h1>
			</div>
		</div>
		<p class="relative mt-3 max-w-md font-semibold text-ink-soft">
			{outOfChallenges ? 'Challenge hari ini sudah selesai. Sampai jumpa besok!' : home.missions.length ? 'Ada misi dari orang tuamu. Yuk selesaikan!' : 'Mau belajar apa hari ini?'}
		</p>
		<div class="relative mt-4 flex flex-wrap gap-2">
			<span class="pill bg-white py-1.5 text-sm text-ink shadow-sm"><Lightning size={16} weight="fill" class="text-primary" aria-hidden="true" />Level {home.child.level}</span>
			<span class="pill bg-white py-1.5 text-sm text-ink shadow-sm"><Star size={16} weight="fill" class="text-sun-deep" aria-hidden="true" />{home.stars} bintang</span>
			{#if home.streak > 0}<span class="pill bg-white py-1.5 text-sm text-ink shadow-sm"><Fire size={16} weight="fill" class="flame text-tangerine" aria-hidden="true" />{home.streak} hari beruntun</span>{/if}
			{#if home.challenges_left_today !== null && !outOfChallenges}<span class="pill bg-white py-1.5 text-sm text-ink-soft shadow-sm">Sisa {home.challenges_left_today} challenge hari ini</span>{/if}
		</div>
	</section>

	<!-- Legendary Archetype Card Banner -->
	<div class="anim-rise mt-5">
		<a
			href="/app/play/{id}/card"
			class="press lift group relative flex items-center justify-between overflow-hidden rounded-[1.6rem] bg-gradient-to-r from-amber-400 via-yellow-400 to-amber-500 p-4 sm:p-5 text-ink shadow-lift"
		>
			<div class="relative z-10 flex items-center gap-3.5">
				<span class="grid size-12 place-items-center rounded-2xl bg-white/90 text-amber-600 shadow-xs">
					<Sparkle size={26} weight="fill" class="anim-wiggle" aria-hidden="true" />
				</span>
				<div>
					<span class="inline-block rounded-full bg-night/80 px-2.5 py-0.5 text-[0.7rem] font-black uppercase tracking-wider text-sun">
						Kartu Legendaris
					</span>
					<h2 class="text-lg font-black sm:text-xl">Lihat Gelar Karakter & Rekapmu!</h2>
					<p class="text-xs font-bold text-ink/75">Buka kartu koleksi, cek tier, dan bagikan ke Story</p>
				</div>
			</div>
			<span class="relative z-10 hidden sm:inline-flex items-center gap-1 rounded-full bg-white px-4 py-2 text-xs font-black shadow-xs">
				Buka Kartu
			</span>
		</a>
	</div>

	{#if notice}<p class="anim-nudge mt-4 rounded-2xl bg-sun-tint p-4 font-bold" role="alert">{notice}</p>{/if}

	<!-- Missions from the parent come first: "what should I do next" (§58). -->
	{#if home.missions.length}
		<section class="mt-7" aria-labelledby="missions-title">
			<h2 id="missions-title" class="flex items-center gap-2 text-xl"><Flag size={24} weight="fill" class="anim-wiggle text-tangerine" aria-hidden="true" />Misi dari orang tua</h2>
			<ul class="mt-3 grid gap-4 {home.missions.length > 1 ? 'sm:grid-cols-2' : ''}">
				{#each home.missions as m, i (m.id)}
					<li class="anim-rise" style="--d:{80 + i * 80}ms">
						<FunCard tone="sun" icon={Flag} iconColor="#c9501c" title={m.skill_name} desc="{m.subject_name}: {m.completed} dari {m.target} challenge"
							cta={starting === m.id ? 'Menyiapkan…' : 'Mulai misi'} disabled={starting !== null || outOfChallenges} onclick={() => start(m.id, { mission_id: m.id })}>
							<span class="block h-2.5 overflow-hidden rounded-full bg-white" aria-hidden="true"><span class="bar-fill block h-full rounded-full bg-tangerine" style="width:{(m.completed / m.target) * 100}%"></span></span>
						</FunCard>
					</li>
				{/each}
			</ul>
		</section>
	{/if}

	<section class="mt-7" aria-labelledby="subjects-title">
		<h2 id="subjects-title" class="text-xl">Pilih pelajaran</h2>
		{#if home.subjects.length === 0}
			<p class="card mt-3 p-6">Belum ada challenge untuk kelasmu. Minta orang tua mengecek kelas di profilmu.</p>
		{:else}
			<ul class="mt-3 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each home.subjects as s, i (s.id)}
					{@const look = looks[s.motif] ?? looks.number}
					<li class="anim-rise" style="--d:{160 + i * 90}ms">
						<FunCard size="lg" tone={look.tone} icon={look.icon} iconColor={look.color} title={s.name}
							cta={starting === 's' + s.id ? 'Menyiapkan…' : 'Mulai main'} disabled={starting !== null || outOfChallenges} onclick={() => start('s' + s.id, { subject_id: s.id })}>
							<span class="flex items-center gap-1.5 text-sm font-bold text-ink-soft">
								{#if s.suggested}<Sparkle size={16} weight="fill" style="color:{look.color}" aria-hidden="true" />Ayo coba yang ini!{:else}<Star size={16} weight="fill" class="text-sun-deep" aria-hidden="true" />{s.stars} bintang terkumpul{/if}
							</span>
						</FunCard>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	<div class="anim-rise mt-7 grid gap-4 sm:grid-cols-2" style="--d:350ms">
		{#if home.next_reward}
			<FunCard tone="rose" icon={Gift} iconColor="#e5487a" title="Hadiah berikutnya: {home.next_reward.name}" desc="{home.next_reward.xp_required - home.child.xp} XP lagi" href="/app/play/{id}/rewards">
				<span class="block h-3 overflow-hidden rounded-full bg-white" aria-hidden="true">
					<span class="bar-fill block h-full rounded-full bg-sun" style="--d:600ms; width:{Math.min(100, (home.child.xp / home.next_reward.xp_required) * 100)}%"></span>
				</span>
			</FunCard>
		{/if}
		<FunCard tone="sky" icon={Fire} iconColor="#ff8a4c" title="{home.streak} hari belajar beruntun"
			desc={home.completed_today > 0 ? 'Hari ini sudah belajar. Keren!' : 'Selesaikan 1 challenge hari ini supaya apinya tetap menyala.'} />
	</div>
{:else}
	<div class="mt-3 space-y-4" aria-busy="true">
		<div class="skeleton h-40 rounded-[1.6rem]"></div>
		<div class="grid gap-4 sm:grid-cols-2"><div class="skeleton h-48 rounded-[1.4rem]"></div><div class="skeleton h-48 rounded-[1.4rem]"></div></div>
	</div>
{/if}
