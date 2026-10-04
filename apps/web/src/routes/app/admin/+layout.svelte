<script lang="ts">
	import Wordmark from '$lib/components/Wordmark.svelte';
	import { session } from '$lib/session.svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import SquaresFour from 'phosphor-svelte/lib/SquaresFour';
	import Question from 'phosphor-svelte/lib/Question';
	import ClipboardText from 'phosphor-svelte/lib/ClipboardText';
	import MagicWand from 'phosphor-svelte/lib/MagicWand';
	import Broadcast from 'phosphor-svelte/lib/Broadcast';
	import GitDiff from 'phosphor-svelte/lib/GitDiff';
	import UsersThree from 'phosphor-svelte/lib/UsersThree';
	import TreeStructure from 'phosphor-svelte/lib/TreeStructure';
	import Images from 'phosphor-svelte/lib/Images';
	import ClockCounterClockwise from 'phosphor-svelte/lib/ClockCounterClockwise';
	import SignOut from 'phosphor-svelte/lib/SignOut';
	let { children } = $props();
	const nav = [
		{ href: '/app/admin', label: 'Ringkasan', icon: SquaresFour },
		{ href: '/app/admin/questions', label: 'Bank soal', icon: Question },
		{ href: '/app/admin/review', label: 'Antrian review', icon: ClipboardText },
		{ href: '/app/admin/content-studio', label: 'Content Studio', icon: MagicWand },
		{ href: '/app/admin/pipeline', label: 'Pipeline', icon: Broadcast },
		{ href: '/app/admin/curriculum-changes', label: 'Perubahan kurikulum', icon: GitDiff },
		{ href: '/app/admin/creators', label: 'Kreator', icon: UsersThree },
		{ href: '/app/admin/curriculum', label: 'Kurikulum', icon: TreeStructure },
		{ href: '/app/admin/assets', label: 'Pustaka aset', icon: Images },
		{ href: '/app/admin/audit', label: 'Log audit', icon: ClockCounterClockwise }
	];
	const active = (h: string) => (h === '/app/admin' ? page.url.pathname === h : page.url.pathname.startsWith(h));
	const roleLabel: Record<string, string> = { admin: 'Admin', reviewer: 'Reviewer', content_editor: 'Editor konten' };
	$effect(() => {
		if (session.user && !session.isStaff) goto('/app/parent', { replaceState: true });
	});
</script>

{#if session.isStaff}
	<div class="min-h-dvh lg:grid lg:grid-cols-[15.5rem_1fr]">
		<aside class="bg-night text-white lg:sticky lg:top-0 lg:flex lg:h-dvh lg:flex-col">
			<div class="flex items-center justify-between gap-3 px-5 py-4 lg:py-6">
				<Wordmark light href="/app/admin" />
				<span class="pill bg-white/10 text-white/80">CMS</span>
			</div>
			<nav aria-label="CMS" class="overflow-x-auto px-3 pb-3 lg:overflow-visible">
				<ul class="flex gap-1 lg:flex-col">
					{#each nav as n}
						<li>
							<a href={n.href} aria-current={active(n.href) ? 'page' : undefined}
								class="flex min-h-11 items-center gap-3 rounded-xl px-3 font-semibold whitespace-nowrap transition-colors {active(n.href) ? 'bg-white/12 text-white' : 'text-white/65 hover:bg-white/6 hover:text-white'}">
								<n.icon size={20} weight={active(n.href) ? 'fill' : 'regular'} class={active(n.href) ? 'text-sun' : ''} aria-hidden="true" />{n.label}
							</a>
						</li>
					{/each}
				</ul>
			</nav>
			<div class="mt-auto hidden items-center gap-3 border-t border-white/10 p-4 lg:flex">
				<span class="grid size-9 shrink-0 place-items-center rounded-full bg-primary text-sm font-extrabold">{session.user?.display_name[0]}</span>
				<div class="min-w-0 flex-1 text-sm leading-tight">
					<p class="truncate font-bold">{session.user?.display_name}</p>
					<p class="text-white/60">{roleLabel[session.user?.role ?? '']}</p>
				</div>
				<button class="grid size-9 place-items-center rounded-full text-white/70 hover:bg-white/10 hover:text-white" aria-label="Keluar"
					onclick={async () => { await session.logout(); goto('/login'); }}><SignOut size={18} weight="bold" aria-hidden="true" /></button>
			</div>
		</aside>
		<main class="min-w-0 px-4 py-6 sm:px-8 lg:py-10">
			<div class="mx-auto max-w-7xl">{@render children()}</div>
		</main>
	</div>
{/if}
