<script lang="ts">
	import Wordmark from '$lib/components/Wordmark.svelte';
	import { session } from '$lib/session.svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import SquaresFour from 'phosphor-svelte/lib/SquaresFour';
	import UsersThree from 'phosphor-svelte/lib/UsersThree';
	import Crown from 'phosphor-svelte/lib/Crown';
	import GearSix from 'phosphor-svelte/lib/GearSix';
	import GameController from 'phosphor-svelte/lib/GameController';
	import SignOut from 'phosphor-svelte/lib/SignOut';
	let { children } = $props();

	const nav = [
		{ href: '/app/parent', label: 'Ringkasan', icon: SquaresFour },
		{ href: '/app/parent/children', label: 'Anak', icon: UsersThree },
		{ href: '/app/parent/subscription', label: 'Langganan', icon: Crown },
		{ href: '/app/parent/settings', label: 'Pengaturan', icon: GearSix }
	];
	const active = (href: string) => (href === '/app/parent' ? page.url.pathname === href : page.url.pathname.startsWith(href));
	async function logout() {
		await session.logout();
		goto('/');
	}
	const initials = $derived((session.user?.display_name ?? '?').split(' ').map((w) => w[0]).slice(0, 2).join('').toUpperCase());
</script>

<div class="min-h-dvh md:grid md:grid-cols-[16rem_1fr]">
	<aside class="sticky top-0 hidden h-dvh flex-col gap-6 border-r border-line bg-paper p-5 md:flex">
		<Wordmark href="/app/parent" />
		<nav aria-label="Area orang tua">
			<ul class="space-y-1">
				{#each nav as n}
					<li>
						<a href={n.href} aria-current={active(n.href) ? 'page' : undefined}
							class="flex min-h-11 items-center gap-3 rounded-xl px-3 font-semibold transition-colors {active(n.href) ? 'bg-primary-tint text-primary-deep' : 'text-ink-soft hover:bg-paper-deep hover:text-ink'}">
							<n.icon size={22} weight={active(n.href) ? 'fill' : 'bold'} aria-hidden="true" />{n.label}
						</a>
					</li>
				{/each}
			</ul>
		</nav>
		<a href="/app/play" class="btn btn-sun"><GameController size={20} weight="fill" aria-hidden="true" />Mode anak</a>
		<div class="mt-auto flex items-center gap-3 rounded-2xl bg-paper-deep p-3">
			<span class="grid size-10 shrink-0 place-items-center rounded-full bg-primary text-sm font-extrabold text-white" aria-hidden="true">{initials}</span>
			<div class="min-w-0 flex-1">
				<p class="truncate text-sm font-bold">{session.user?.display_name}</p>
				<p class="truncate text-xs text-ink-soft">{session.user?.email}</p>
			</div>
			<button class="grid size-10 place-items-center rounded-full text-ink-soft hover:bg-paper hover:text-ink" onclick={logout} aria-label="Keluar"><SignOut size={20} weight="bold" aria-hidden="true" /></button>
		</div>
	</aside>

	<div class="min-w-0">
		<header class="sticky top-0 z-20 flex items-center justify-between border-b border-line bg-paper/95 px-4 py-2 backdrop-blur md:hidden">
			<Wordmark href="/app/parent" />
			<a href="/app/play" class="btn btn-sun btn-sm"><GameController size={18} weight="fill" aria-hidden="true" />Mode anak</a>
		</header>
		<main class="mx-auto max-w-6xl px-4 pt-6 pb-28 sm:px-6 md:pt-10 md:pb-16">
			{@render children()}
		</main>
	</div>

	<nav class="safe-bottom fixed inset-x-0 bottom-0 z-20 border-t border-line bg-paper md:hidden" aria-label="Area orang tua">
		<ul class="grid grid-cols-4">
			{#each nav as n}
				<li>
					<a href={n.href} aria-current={active(n.href) ? 'page' : undefined}
						class="flex min-h-14 flex-col items-center justify-center gap-0.5 pt-1.5 text-xs font-bold {active(n.href) ? 'text-primary-deep' : 'text-ink-soft'}">
						<n.icon size={24} weight={active(n.href) ? 'fill' : 'regular'} aria-hidden="true" />{n.label}
					</a>
				</li>
			{/each}
		</ul>
	</nav>
</div>
