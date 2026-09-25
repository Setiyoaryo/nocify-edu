<script lang="ts">
	import { get } from '$lib/api';
	import { relative } from '$lib/format';
	import Avatar from '$lib/components/Avatar.svelte';
	import UserPlus from 'phosphor-svelte/lib/UserPlus';
	import Plus from 'phosphor-svelte/lib/Plus';
	import LockKey from 'phosphor-svelte/lib/LockKey';
	import type { Child, Entitlement } from '$lib/types';

	let children = $state<Child[] | null>(null);
	let sub = $state<Entitlement | null>(null);
	Promise.all([get('/children'), get('/subscription')]).then(([c, s]) => {
		children = c.children;
		sub = s.subscription;
	});
	const tones = ['pastel-grape', 'pastel-peach', 'pastel-mint', 'pastel-sky', 'pastel-rose'];
	const full = $derived(!!children && !!sub && children.length >= sub.limits.max_children);
</script>

<svelte:head><title>Anak | EduPlay</title></svelte:head>

<div class="flex flex-wrap items-center justify-between gap-4">
	<h1 class="text-3xl sm:text-4xl">Profil anak</h1>
	{#if children && !full}<a href="/app/parent/children/new" class="btn btn-primary"><Plus size={18} weight="bold" aria-hidden="true" />Tambah anak</a>{/if}
</div>

{#if !children}
	<div class="card skeleton mt-8 h-40" aria-busy="true"></div>
{:else}
	<ul class="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
		{#each children as c, i (c.id)}
			<li class="anim-rise" style="--d:{i * 80}ms">
				<a href="/app/parent/children/{c.id}" class="{tones[i % tones.length]} press lift flex h-full flex-col rounded-[1.4rem] p-5">
					<div class="flex items-center gap-4">
						<Avatar code={c.avatar} size={60} ring />
						<span class="min-w-0 flex-1">
							<span class="block truncate text-xl font-extrabold">{c.nickname}</span>
							<span class="block text-sm font-semibold text-ink-soft">{c.grade_name}, level {c.level}</span>
						</span>
					</div>
					<p class="mt-4 text-sm font-semibold text-ink-soft">Aktif {relative(c.last_activity).toLowerCase()}</p>
					<span class="pill-action mt-4 w-full">Lihat laporan</span>
				</a>
			</li>
		{:else}
			<li class="card p-6 text-ink-soft sm:col-span-2">Belum ada profil anak.</li>
		{/each}
		{#if !full}
			<li class="anim-rise" style="--d:{children.length * 80}ms">
				<a href="/app/parent/children/new" class="press flex h-full min-h-44 flex-col items-center justify-center gap-2 rounded-[1.4rem] border-2 border-dashed border-line p-5 text-ink-soft hover:border-primary hover:text-primary-deep">
					<UserPlus size={32} weight="duotone" aria-hidden="true" /><span class="font-extrabold">Tambah anak</span>
				</a>
			</li>
		{/if}
	</ul>
	{#if full && sub && !sub.premium}
		<div class="mt-6 flex flex-col gap-3 rounded-2xl bg-paper p-5 sm:flex-row sm:items-center">
			<span class="grid size-10 shrink-0 place-items-center rounded-xl bg-sun-tint text-ink"><LockKey size={22} weight="duotone" aria-hidden="true" /></span>
			<p class="flex-1">Paket gratis untuk 1 anak. Berlangganan untuk menambah hingga 5 profil.</p>
			<a class="btn btn-sun btn-sm" href="/app/parent/subscription">Lihat paket</a>
		</div>
	{:else if full}
		<p class="mt-6 text-ink-soft">Batas {sub?.limits.max_children} profil anak tercapai.</p>
	{/if}
{/if}
