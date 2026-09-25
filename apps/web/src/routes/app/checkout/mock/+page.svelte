<script lang="ts">
	// Stand-in for the provider's hosted checkout when PAYMENT_PROVIDER=mock
	// (automated tests/CI only; never production). It asks the API to send
	// itself a signed webhook; the webhook, not this page, activates the plan.
	import { post, ApiError } from '$lib/api';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	const ref = page.url.searchParams.get('ref') ?? '';
	let msg = $state('');
	let busy = $state(false);
	async function pay(outcome: 'success' | 'failed') {
		busy = true;
		try {
			const r = await post(`/dev/mock-checkout/${encodeURIComponent(ref)}/pay`, { outcome });
			if (outcome === 'success') goto(`/app/parent/subscription?checkout=${encodeURIComponent(ref)}`);
			else msg = `Webhook diproses: ${r.result}. Pembayaran ditandai gagal.`;
		} catch (e) {
			msg = e instanceof ApiError ? e.message : 'Gagal.';
		}
		busy = false;
	}
</script>

<main class="mx-auto max-w-md px-4 py-16">
	<div class="card p-8">
		<p class="pill bg-sun-tint text-ink">Pembayaran uji (mode test)</p>
		<h1 class="mt-5 text-3xl">Pembayaran</h1>
		<p class="mt-2 break-all text-ink-soft">Referensi {ref}</p>
		<div class="mt-8 flex flex-wrap gap-3">
			<button class="btn btn-primary" disabled={busy} onclick={() => pay('success')}>Simulasikan pembayaran berhasil</button>
			<button class="btn btn-quiet" disabled={busy} onclick={() => pay('failed')}>Simulasikan gagal</button>
		</div>
		{#if msg}<p class="mt-6" role="status">{msg}</p>{/if}
	</div>
</main>
