<script lang="ts">
	import { get } from '$lib/api';
	import { session } from '$lib/session.svelte';
	import StatusPill from '$lib/components/StatusPill.svelte';
	import Users from 'phosphor-svelte/lib/Users';
	import Baby from 'phosphor-svelte/lib/Baby';
	import UserCheck from 'phosphor-svelte/lib/UserCheck';
	import Crown from 'phosphor-svelte/lib/Crown';
	import Play from 'phosphor-svelte/lib/Play';
	import CheckSquare from 'phosphor-svelte/lib/CheckSquare';
	import CreditCard from 'phosphor-svelte/lib/CreditCard';
	import Warning from 'phosphor-svelte/lib/Warning';
	import CountUp from '$lib/components/CountUp.svelte';
	let s = $state<any>(null);
	get('/admin/stats').then((r) => (s = r));
	const qOrder = ['draft', 'review', 'approved', 'published', 'retired'];
	const aOrder = ['imported', 'license_verified', 'design_approved', 'performance_checked', 'published', 'rejected'];
	const kpis = $derived(s ? [
		{ label: 'Orang tua', value: s.parents, icon: Users, tint: 'bg-primary-tint text-primary-deep' },
		{ label: 'Profil anak', value: s.children, icon: Baby, tint: 'bg-sky-tint text-sky' },
		{ label: 'Aktif (≥1 challenge)', value: s.activated_parents, icon: UserCheck, tint: 'bg-leaf-tint text-leaf-deep' },
		{ label: 'Pelanggan aktif', value: s.active_subscribers, icon: Crown, tint: 'bg-sun-tint text-ink' },
		{ label: 'Challenge dimulai 7h', value: s.challenges_started_7d, icon: Play, tint: 'bg-primary-tint text-primary-deep' },
		{ label: 'Challenge selesai 7h', value: s.challenges_completed_7d, icon: CheckSquare, tint: 'bg-leaf-tint text-leaf-deep' },
		{ label: 'Pembayaran gagal 7h', value: s.payment_failures_7d, icon: CreditCard, tint: 'bg-tangerine-tint text-tangerine-deep' },
		{ label: 'Webhook error', value: s.webhook_errors, icon: Warning, tint: s.webhook_errors ? 'bg-rose-tint text-[#a8264f]' : 'bg-paper-deep text-ink-soft' }
	] : []);
	const completion = $derived(s && s.challenges_started_7d ? Math.round((s.challenges_completed_7d / s.challenges_started_7d) * 100) : null);
</script>

<svelte:head><title>Ringkasan | CMS EduPlay</title></svelte:head>
<p class="font-semibold text-ink-soft">Halo, {session.user?.display_name}</p>
<h1 class="text-3xl sm:text-4xl">Ringkasan</h1>

{#if s}
	<div class="mt-8 grid grid-cols-2 gap-4 lg:grid-cols-4">
		{#each kpis as k, i}
			<div class="card anim-rise flex items-center gap-4 p-5" style="--d:{i * 50}ms">
				<span class="grid size-12 shrink-0 place-items-center rounded-2xl {k.tint}"><k.icon size={26} weight="duotone" aria-hidden="true" /></span>
				<div class="min-w-0">
					<p class="text-sm leading-snug font-semibold text-ink-soft">{k.label}</p>
					<p class="text-3xl leading-tight font-extrabold"><CountUp value={k.value} duration={700} /></p>
				</div>
			</div>
		{/each}
	</div>
	{#if completion !== null}<p class="mt-3 text-sm text-ink-soft">Tingkat penyelesaian challenge 7 hari: <b class="text-ink">{completion}%</b></p>{/if}

	<div class="anim-rise mt-8 grid gap-6 lg:grid-cols-2" style="--d:300ms">
		<section class="card p-6">
			<div class="flex items-center justify-between"><h2 class="text-xl">Bank soal</h2><a href="/app/admin/questions" class="flex items-center gap-1 text-sm font-bold text-primary-deep hover:underline">Kelola</a></div>
			<ul class="mt-4 divide-y divide-line">
				{#each qOrder as k}
					<li><a class="flex min-h-12 items-center justify-between rounded-lg px-2 hover:bg-paper-deep" href="/app/admin/questions?status={k}"><StatusPill status={k} /><b class="tabular-nums">{s.questions[k] ?? 0}</b></a></li>
				{/each}
			</ul>
		</section>
		<section class="card p-6">
			<div class="flex items-center justify-between"><h2 class="text-xl">Pustaka aset</h2><a href="/app/admin/assets" class="flex items-center gap-1 text-sm font-bold text-primary-deep hover:underline">Kelola</a></div>
			<ul class="mt-4 divide-y divide-line">
				{#each aOrder as k}
					<li class="flex min-h-12 items-center justify-between px-2"><StatusPill status={k} /><b class="tabular-nums">{s.assets[k] ?? 0}</b></li>
				{/each}
			</ul>
		</section>
	</div>
{:else}
	<div class="mt-8 grid grid-cols-2 gap-4 lg:grid-cols-4">{#each Array(8) as _}<div class="card skeleton h-24"></div>{/each}</div>
{/if}
