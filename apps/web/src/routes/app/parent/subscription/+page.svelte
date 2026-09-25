<script lang="ts">
	import { get, post, ApiError } from '$lib/api';
	import { date, rupiah } from '$lib/format';
	import { plans } from '$lib/pricing';
	import { page } from '$app/state';
	import { onDestroy } from 'svelte';
	import payment from '$lib/illustrations/payment.svg';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	import Crown from 'phosphor-svelte/lib/Crown';
	import ShieldCheck from 'phosphor-svelte/lib/ShieldCheck';
	import HourglassMedium from 'phosphor-svelte/lib/HourglassMedium';
	import WarningCircle from 'phosphor-svelte/lib/WarningCircle';
	import type { Entitlement } from '$lib/types';

	let sub = $state<Entitlement | null>(null);
	let offer = $state<{ amount: number } | null>(null);
	let busy = $state(false);
	let error = $state('');
	// Returning from Midtrans: the webhook, not this redirect, activates the plan (§22).
	const returning = page.url.searchParams.get('checkout');
	let payState = $state<'waiting' | 'paid' | 'failed' | 'timeout' | null>(returning ? 'waiting' : null);
	let timer: ReturnType<typeof setTimeout>;

	async function load() {
		const r = await get('/subscription');
		sub = r.subscription;
		offer = r.offer;
	}
	async function poll(tries = 0) {
		try {
			const { status } = await get(`/subscription/checkout/${encodeURIComponent(returning!)}`);
			if (status === 'paid') {
				payState = 'paid';
				return load();
			}
			if (status === 'failed' || status === 'expired') {
				payState = 'failed';
				return;
			}
		} catch {}
		if (tries > 40) payState = 'timeout';
		else timer = setTimeout(() => poll(tries + 1), 3000);
	}
	load();
	if (returning) poll();
	onDestroy(() => clearTimeout(timer));

	async function checkout() {
		busy = true;
		error = '';
		try {
			const r = await post('/subscription/checkout', { plan: 'monthly' });
			location.href = r.redirect_url;
		} catch (e) {
			error = e instanceof ApiError ? e.message : 'Gagal memulai pembayaran.';
			busy = false;
		}
	}
	const statusLabel: Record<string, string> = { active: 'Aktif', trial: 'Masa coba', past_due: 'Menunggu perpanjangan', cancelled: 'Tidak diperpanjang', expired: 'Berakhir' };
</script>

<svelte:head><title>Langganan | EduPlay</title></svelte:head>

<h1 class="text-3xl sm:text-4xl">Langganan</h1>

{#if payState === 'waiting'}
	<div class="mt-6 flex items-center gap-3 rounded-2xl bg-sky-tint p-4" role="status">
		<HourglassMedium size={24} weight="duotone" class="shrink-0 animate-spin text-sky [animation-duration:2.4s]" aria-hidden="true" />
		<p>Menunggu konfirmasi pembayaran dari Midtrans. Halaman ini diperbarui otomatis.</p>
	</div>
{:else if payState === 'paid'}
	<div class="anim-pop mt-6 flex items-center gap-3 rounded-2xl bg-leaf-tint p-4 text-leaf-deep" role="status">
		<CheckCircle size={24} weight="fill" class="shrink-0" aria-hidden="true" /><p class="font-bold">Pembayaran diterima. Langganan Anda aktif.</p>
	</div>
{:else if payState === 'failed'}
	<div class="mt-6 flex items-center gap-3 rounded-2xl bg-tangerine-tint p-4 text-tangerine-deep" role="alert">
		<WarningCircle size={24} weight="fill" class="shrink-0" aria-hidden="true" /><p class="font-bold">Pembayaran tidak berhasil atau kedaluwarsa. Silakan coba lagi.</p>
	</div>
{:else if payState === 'timeout'}
	<div class="mt-6 rounded-2xl bg-sun-tint p-4" role="status">Konfirmasi pembayaran belum kami terima. Jika Anda sudah membayar, langganan aktif otomatis dalam beberapa menit; muat ulang halaman ini nanti.</div>
{/if}

{#if !sub || !offer}
	<div class="card skeleton mt-6 h-64" aria-busy="true"></div>
{:else if sub.premium}
	<section class="card anim-rise mt-6 grid items-center gap-6 p-6 sm:p-8 md:grid-cols-[1fr_auto]">
		<div>
			<span class="pill bg-sun text-ink"><Crown size={16} weight="fill" aria-hidden="true" />Paket Bulanan</span>
			<p class="mt-3 text-2xl font-extrabold">{statusLabel[sub.status ?? ''] ?? sub.status}</p>
			{#if sub.current_period_end}
				<p class="mt-1 text-ink-soft">
					{sub.cancel_at_period_end ? 'Akses berakhir pada' : sub.status === 'past_due' ? 'Masa aktif berakhir pada' : 'Berlaku sampai'}
					<b class="text-ink">{date(sub.current_period_end)}</b>.
				</p>
			{/if}
			<p class="mt-3 max-w-lg text-sm text-ink-soft">Satu pembayaran berlaku satu bulan. Bayar lagi kapan saja untuk menambah satu bulan dari tanggal berakhir.</p>
			<div class="mt-6 flex flex-wrap gap-3">
				<button class="btn btn-primary" disabled={busy} onclick={checkout}>Perpanjang 1 bulan</button>
			</div>
			{#if error}<p class="field-error mt-3" role="alert">{error}</p>{/if}
		</div>
		<img src={payment} alt="" class="mx-auto hidden w-44 md:block" />
	</section>
{:else}
	<section class="anim-rise mt-6 grid gap-6 lg:grid-cols-[1.2fr_1fr]">
		<div class="card relative p-6 ring-2 ring-primary sm:p-8">
			{#if sub.status === 'expired'}<p class="mb-3 text-sm font-semibold text-tangerine-deep">Langganan sebelumnya sudah berakhir.</p>{/if}
			<span class="pill bg-sun text-ink"><Crown size={16} weight="fill" aria-hidden="true" />Bulanan</span>
			<p class="mt-4"><span class="text-4xl font-extrabold">{rupiah(offer.amount)}</span> <span class="font-semibold text-ink-soft">/ bulan</span></p>
			<ul class="mt-6 space-y-3">
				{#each plans.monthly as f}<li class="flex gap-3"><CheckCircle size={22} weight="fill" class="shrink-0 text-primary" aria-hidden="true" />{f}</li>{/each}
			</ul>
			{#if error}<p class="field-error mt-4" role="alert">{error}</p>{/if}
			<button class="btn btn-primary mt-8 w-full" disabled={busy} onclick={checkout}>{busy ? 'Menyiapkan pembayaran…' : 'Berlangganan sekarang'}</button>
		</div>
		<aside class="card flex flex-col gap-4 p-6 sm:p-8">
			<img src={payment} alt="" class="mx-auto w-40" />
			<p class="flex items-start gap-3 text-sm text-ink-soft"><ShieldCheck size={22} weight="duotone" class="shrink-0 text-leaf-deep" aria-hidden="true" />Pembayaran diproses aman oleh Midtrans: QRIS, virtual account bank, e-wallet, atau kartu.</p>
			<p class="text-sm text-ink-soft">Langganan aktif otomatis setelah pembayaran dikonfirmasi. Tidak ada perpanjangan otomatis: Anda yang memutuskan kapan membayar lagi.</p>
		</aside>
	</section>
{/if}
