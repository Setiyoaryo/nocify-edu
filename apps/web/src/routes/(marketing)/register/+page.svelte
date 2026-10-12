<script lang="ts">
	import Meta from '$lib/components/Meta.svelte';
	import AuthShell from '$lib/components/AuthShell.svelte';
	import { post, ApiError } from '$lib/api';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import art from '$lib/illustrations/parent-child.svg';

	let googleEnabled = $state(false);
	let ready = $state(false);
	onMount(async () => {
		ready = true;
		try {
			const response = await fetch('/api/v1/auth/google/status');
			googleEnabled = response.ok && (await response.json()).enabled === true;
		} catch { googleEnabled = false; }
	});

	let form = $state({ display_name: '', email: '', password: '', consent: false });
	let fields = $state<Record<string, string>>({});
	let error = $state('');
	let busy = $state(false);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = '';
		fields = {};
		try {
			await post('/auth/register', form);
			await goto('/app/parent/children/new?welcome=1');
		} catch (err) {
			if (err instanceof ApiError) {
				fields = err.fields;
				if (!Object.keys(err.fields).length) error = err.message;
			} else error = 'Gagal mendaftar.';
		} finally {
			busy = false;
		}
	}
</script>

<Meta title="Daftar" description="Buat akun orang tua EduPlay secara gratis." />

<AuthShell title="Buat akun orang tua" lead="Gratis untuk 1 anak. Profil anak dibuat setelah ini." image={art}
	points={['Gratis untuk 1 anak, tanpa kartu kredit', 'Challenge 5–10 menit sesuai kelas', 'Laporan yang menjawab: sudah bisa apa, perlu latihan apa']}>
	{#if googleEnabled}
		<a class="btn btn-primary mt-8 flex w-full items-center justify-center" href="/api/v1/auth/google/start">Daftar dengan Google</a>
		<p class="mt-2 text-center text-xs text-ink-soft">Persetujuan orang tua akan diminta sebelum akun dibuat.</p>
	{/if}
	<form class="mt-8 space-y-5" method="post" onsubmit={submit} novalidate>
		<div>
			<label class="label" for="name">Nama Anda</label>
			<input class="field" id="name" autocomplete="name" bind:value={form.display_name} aria-invalid={!!fields.display_name} aria-describedby="name-err" />
			{#if fields.display_name}<p id="name-err" class="field-error">{fields.display_name}</p>{/if}
		</div>
		<div>
			<label class="label" for="email">Email</label>
			<input class="field" id="email" type="email" autocomplete="email" bind:value={form.email} aria-invalid={!!fields.email} aria-describedby="email-err" />
			{#if fields.email}<p id="email-err" class="field-error">{fields.email}</p>{/if}
		</div>
		<div>
			<label class="label" for="password">Kata sandi</label>
			<input class="field" id="password" type="password" autocomplete="new-password" bind:value={form.password} aria-invalid={!!fields.password} aria-describedby="pw-hint" />
			<p id="pw-hint" class={fields.password ? 'field-error' : 'mt-1 text-sm text-ink-soft'}>{fields.password ?? 'Minimal 8 karakter.'}</p>
		</div>
		<div class="rounded-2xl bg-paper-deep p-4">
			<label class="flex cursor-pointer gap-3 text-sm">
				<input type="checkbox" class="mt-0.5 size-5 shrink-0 accent-primary" bind:checked={form.consent} aria-invalid={!!fields.consent} />
				<span>
					Saya orang tua/wali anak yang akan menggunakan EduPlay dan menyetujui pemrosesan data anak untuk keperluan belajarnya sesuai <a href="/privacy" target="_blank" class="text-primary-deep underline">Kebijakan Privasi</a> dan <a href="/terms" target="_blank" class="text-primary-deep underline">Syarat Layanan</a>.
				</span>
			</label>
			{#if fields.consent}<p class="field-error">{fields.consent}</p>{/if}
		</div>
		{#if error}<p class="field-error" role="alert">{error}</p>{/if}
		<button class="btn btn-primary w-full" disabled={busy || !ready}>{busy ? 'Memproses…' : 'Buat akun'}</button>
	</form>
	<p class="mt-6 text-center text-ink-soft">Sudah punya akun? <a class="font-bold text-primary-deep hover:underline" href="/login">Masuk</a></p>
</AuthShell>
