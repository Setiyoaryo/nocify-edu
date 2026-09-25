<script lang="ts">
	import Meta from '$lib/components/Meta.svelte';
	import AuthShell from '$lib/components/AuthShell.svelte';
	import { post, ApiError } from '$lib/api';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import loginArt from '$lib/illustrations/login.svg';
	import type { User } from '$lib/types';

	// Prerendered page: keep submit disabled until hydrated so an early tap never
	// falls back to a native form submit.
	let ready = $state(false);
	onMount(() => (ready = true));

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let busy = $state(false);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			const { user } = await post<{ user: User }>('/auth/login', { email, password });
			const next = new URLSearchParams(location.search).get('next');
			await goto(next?.startsWith('/app') ? next : user.role === 'parent' ? '/app/parent' : '/app/admin');
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Gagal masuk.';
		} finally {
			busy = false;
		}
	}
</script>

<Meta title="Masuk" description="Masuk ke akun orang tua EduPlay." />

<AuthShell title="Selamat datang kembali" lead="Masuk untuk melihat perkembangan anak dan melanjutkan belajar." image={loginArt}
	points={['Laporan per skill untuk setiap anak', 'Misi latihan yang bisa Anda tentukan', 'Aman: tanpa iklan untuk anak']}>
	<form class="mt-8 space-y-5" method="post" onsubmit={submit} novalidate>
		<div>
			<label class="label" for="email">Email</label>
			<input class="field" id="email" type="email" autocomplete="email" required bind:value={email} />
		</div>
		<div>
			<label class="label" for="password">Kata sandi</label>
			<input class="field" id="password" type="password" autocomplete="current-password" required bind:value={password} />
		</div>
		{#if error}<p class="field-error" role="alert">{error}</p>{/if}
		<button class="btn btn-primary w-full" disabled={busy || !ready}>{busy ? 'Memproses…' : 'Masuk'}</button>
	</form>
	<p class="mt-6 text-center text-ink-soft">Belum punya akun? <a class="font-bold text-primary-deep hover:underline" href="/register">Daftar gratis</a></p>
</AuthShell>
