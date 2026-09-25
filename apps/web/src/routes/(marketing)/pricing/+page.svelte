<script lang="ts">
	import Meta from '$lib/components/Meta.svelte';
	import PageHero from '$lib/components/PageHero.svelte';
	import CtaBand from '$lib/components/CtaBand.svelte';
	import { monthlyPriceLabel, plans } from '$lib/pricing';
	import { track } from '$lib/api';
	import { onMount } from 'svelte';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	import Crown from 'phosphor-svelte/lib/Crown';
	import ShieldCheck from 'phosphor-svelte/lib/ShieldCheck';
	onMount(() => track('pricing_viewed'));
</script>

<Meta title="Harga" description="Gratis untuk 1 anak. Langganan bulanan untuk challenge tanpa batas, laporan per skill lengkap, dan misi latihan." />
<PageHero title="Harga yang sederhana" lead="Mulai gratis. Berlangganan saat laporan per skill mulai berguna bagi Anda. Bisa berhenti kapan saja." />

<div class="mx-auto -mt-2 max-w-4xl px-4 sm:px-6">
	<div class="grid gap-6 md:grid-cols-2">
		<section class="card flex flex-col p-7" aria-labelledby="free">
			<h2 id="free" class="text-2xl">Gratis</h2>
			<p class="mt-1 text-ink-soft">Untuk mencoba bersama 1 anak</p>
			<p class="mt-5 text-4xl font-extrabold">Rp0</p>
			<ul class="mt-6 flex-1 space-y-3">
				{#each plans.free as f}<li class="flex gap-3"><CheckCircle size={22} weight="fill" class="shrink-0 text-leaf" aria-hidden="true" />{f}</li>{/each}
			</ul>
			<a href="/register" class="btn btn-quiet mt-8 w-full">Daftar gratis</a>
		</section>
		<section class="card relative flex flex-col p-7 ring-2 ring-primary" aria-labelledby="monthly">
			<span class="pill absolute -top-3.5 left-7 bg-sun text-ink"><Crown size={16} weight="fill" aria-hidden="true" />Paling lengkap</span>
			<h2 id="monthly" class="text-2xl">Bulanan</h2>
			<p class="mt-1 text-ink-soft">Untuk keluarga yang ingin memantau rutin</p>
			<p class="mt-5"><span class="text-4xl font-extrabold">{monthlyPriceLabel}</span> <span class="font-semibold text-ink-soft">/ bulan</span></p>
			<ul class="mt-6 flex-1 space-y-3">
				{#each plans.monthly as f}<li class="flex gap-3"><CheckCircle size={22} weight="fill" class="shrink-0 text-primary" aria-hidden="true" />{f}</li>{/each}
			</ul>
			<a href="/register" class="btn btn-primary mt-8 w-full">Mulai gratis, lalu berlangganan</a>
		</section>
	</div>
	<p class="mt-8 flex items-start gap-3 rounded-2xl bg-paper p-5 text-sm text-ink-soft">
		<ShieldCheck size={22} weight="duotone" class="shrink-0 text-leaf-deep" aria-hidden="true" />
		Pembayaran diproses aman oleh Midtrans (QRIS, virtual account, e-wallet, kartu). Satu pembayaran berlaku 1 bulan, tanpa perpanjangan otomatis.
	</p>
</div>

<CtaBand title="Masih ragu? Mulai dari yang gratis" body="Coba bersama 1 anak dulu. Berlangganan hanya jika laporannya terasa berguna." />
