<script lang="ts">
	import { get, post, ApiError } from '$lib/api';
	import { speech } from '$lib/speech.svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import ParentGate from '$lib/components/ParentGate.svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import Star from 'phosphor-svelte/lib/Star';
	import Fire from 'phosphor-svelte/lib/Fire';
	import SpeakerHigh from 'phosphor-svelte/lib/SpeakerHigh';
	import SpeakerSlash from 'phosphor-svelte/lib/SpeakerSlash';
	import SignOut from 'phosphor-svelte/lib/SignOut';
	import House from 'phosphor-svelte/lib/House';
	import Gift from 'phosphor-svelte/lib/Gift';
	import UserCircle from 'phosphor-svelte/lib/UserCircle';
	import Backspace from 'phosphor-svelte/lib/Backspace';
	import type { Child } from '$lib/types';
	import { setContext } from 'svelte';

	let { children } = $props();
	const id = page.params.childId as string;
	const ctx = $state<{ child: Child | null; stars: number; streak: number; refresh: () => Promise<void> }>({ child: null, stars: 0, streak: 0, refresh });
	setContext('play', ctx);

	async function refresh() {
		const home = await get(`/children/${id}/home`);
		ctx.child = home.child;
		ctx.stars = home.stars;
		ctx.streak = home.streak;
	}

	const key = `eduplay.pin.${id}`;
	let unlocked = $state(false);
	let pin = $state('');
	let pinError = $state('');
	let gate = $state(false);

	refresh().then(() => {
		let ok = false;
		try {
			ok = sessionStorage.getItem(key) === '1';
		} catch {}
		unlocked = !ctx.child?.has_pin || ok;
	});

	async function press(d: string) {
		pinError = '';
		pin = d === 'hapus' ? pin.slice(0, -1) : (pin + d).slice(0, 4);
		if (pin.length === 4) {
			try {
				await post(`/children/${id}/verify-pin`, { pin });
				try {
					sessionStorage.setItem(key, '1');
				} catch {}
				unlocked = true;
			} catch (e) {
				pinError = e instanceof ApiError ? e.message : 'Coba lagi.';
				pin = '';
			}
		}
	}

	const inChallenge = $derived(page.url.pathname.includes('/challenge/'));
	const tabs = [
		{ href: `/app/play/${id}`, label: 'Main', icon: House },
		{ href: `/app/play/${id}/rewards`, label: 'Hadiah', icon: Gift },
		{ href: `/app/play/${id}/profile`, label: 'Profil', icon: UserCircle }
	];
</script>

{#if !ctx.child}
	<p class="grid min-h-dvh place-items-center text-ink-soft">Memuat…</p>
{:else if !unlocked}
	<main class="flex min-h-dvh flex-col items-center justify-center bg-primary px-4 text-white">
		<Avatar code={ctx.child.avatar} size={96} ring />
		<h1 class="mt-4 text-3xl">Halo, {ctx.child.nickname}!</h1>
		<p class="mt-1 text-white/80">Masukkan PIN kamu</p>
		<p class="mt-5 flex gap-3" aria-label="{pin.length} dari 4 angka">
			{#each Array(4) as _, i}<span class="size-4 rounded-full {i < pin.length ? 'bg-sun' : 'bg-white/30'}"></span>{/each}
		</p>
		<p class="min-h-6 font-semibold text-sun" role="alert">{pinError}</p>
		<div class="mt-2 grid grid-cols-3 gap-3">
			{#each ['1', '2', '3', '4', '5', '6', '7', '8', '9', '', '0', 'hapus'] as d}
				{#if d}
					<button class="tile grid size-20 place-items-center text-3xl text-ink" onclick={() => press(d)} aria-label={d === 'hapus' ? 'Hapus' : d}>
						{#if d === 'hapus'}<Backspace size={28} weight="bold" aria-hidden="true" />{:else}{d}{/if}
					</button>
				{:else}<span></span>{/if}
			{/each}
		</div>
		<a href="/app/play" class="mt-8 font-semibold text-white/80 underline">Bukan aku</a>
	</main>
{:else}
	<div class="flex min-h-dvh flex-col">
		<header class="sticky top-0 z-20 bg-canvas/95 backdrop-blur">
			<div class="mx-auto flex max-w-4xl items-center gap-2 px-3 pt-[max(0.75rem,env(safe-area-inset-top))] pb-2 sm:gap-3 sm:px-4">
				<Avatar code={ctx.child.avatar} size={46} />
				<div class="min-w-0 flex-1 leading-tight">
					<p class="truncate text-lg font-extrabold">{ctx.child.nickname}</p>
					<p class="text-sm font-semibold whitespace-nowrap text-ink-soft">Level {ctx.child.level}</p>
				</div>
				{#if ctx.streak > 0}
					<p class="pill hidden bg-tangerine-tint py-1.5 text-base text-tangerine-deep min-[400px]:inline-flex" aria-label="{ctx.streak} hari beruntun"><Fire size={20} weight="fill" aria-hidden="true" />{ctx.streak}</p>
				{/if}
				{#key ctx.stars}<p class="pill anim-pop bg-sun-tint py-1.5 text-base text-ink" aria-label="{ctx.stars} bintang"><Star size={20} weight="fill" class="text-sun-deep" aria-hidden="true" />{ctx.stars}</p>{/key}
				<button class="tile grid size-11 shrink-0 place-items-center text-primary sm:size-12" onclick={() => speech.toggleMute()} aria-pressed={speech.muted} aria-label={speech.muted ? 'Nyalakan suara' : 'Matikan suara'}>
					{#if speech.muted}<SpeakerSlash size={22} weight="bold" aria-hidden="true" />{:else}<SpeakerHigh size={22} weight="fill" aria-hidden="true" />{/if}
				</button>
				{#if !inChallenge}
					<button class="tile grid size-11 shrink-0 place-items-center text-ink-soft sm:size-12" onclick={() => (gate = true)} aria-label="Keluar ke area orang tua"><SignOut size={22} weight="bold" aria-hidden="true" /></button>
				{/if}
			</div>
		</header>

		<main class="mx-auto w-full max-w-4xl flex-1 px-4 {inChallenge ? 'pb-6' : 'pb-32'}">
			{@render children()}
		</main>

		{#if !inChallenge}
			<nav class="safe-bottom fixed inset-x-0 bottom-0 z-20 border-t border-line bg-paper" aria-label="Menu anak">
				<ul class="mx-auto grid max-w-md grid-cols-3">
					{#each tabs as t}
						{@const active = page.url.pathname === t.href}
						<li>
							<a href={t.href} aria-current={active ? 'page' : undefined} class="flex min-h-16 flex-col items-center justify-center gap-0.5 font-extrabold {active ? 'text-primary' : 'text-ink-soft'}">
								<t.icon size={28} weight={active ? 'fill' : 'regular'} aria-hidden="true" />{t.label}
							</a>
						</li>
					{/each}
				</ul>
			</nav>
		{/if}
	</div>
	{#if gate}
		<ParentGate onpass={() => { speech.stop(); goto('/app/parent'); }} oncancel={() => (gate = false)} />
	{/if}
{/if}
