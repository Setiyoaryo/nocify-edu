<script lang="ts">
	// Hidden staff entry point. Deliberately not linked from anywhere and
	// excluded from search engines; staff reach it by typing the URL.
	// Hiding is obscurity, not access control: the API still enforces the role.
	import Wordmark from '$lib/components/Wordmark.svelte';
	import { post, ApiError } from '$lib/api';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import type { User } from '$lib/types';
	import LockKey from 'phosphor-svelte/lib/LockKey';

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
			if (user.role === 'parent') {
				await post('/auth/logout').catch(() => {});
				error = 'Akun ini bukan akun pengelola.';
				return;
			}
			const next = page.url.searchParams.get('next');
			await goto(next?.startsWith('/app') ? next : '/app/admin');
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Gagal masuk. Periksa email dan kata sandi.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head>
	<meta name="robots" content="noindex, nofollow" />
	<title>Masuk Pengelola | EduPlay</title>
</svelte:head>

<div class="grid min-h-dvh place-items-center bg-paper-deep px-4 py-10">
	<div class="w-full max-w-sm">
		<div class="flex justify-center"><Wordmark /></div>
		<form class="card mt-6 space-y-4 p-5 sm:p-6" onsubmit={submit} novalidate>
			<div class="flex items-center gap-2.5">
				<span class="grid size-9 shrink-0 place-items-center rounded-xl bg-primary-tint text-primary-deep">
					<LockKey size={20} weight="fill" aria-hidden="true" />
				</span>
				<h1 class="text-lg font-extrabold text-ink">Masuk Pengelola</h1>
			</div>
			<div>
				<label class="label text-xs" for="email">Email</label>
				<input class="field text-sm" id="email" type="email" autocomplete="email" required bind:value={email} />
			</div>
			<div>
				<label class="label text-xs" for="password">Kata sandi</label>
				<input class="field text-sm" id="password" type="password" autocomplete="current-password" required bind:value={password} />
			</div>
			{#if error}<p class="field-error" role="alert">{error}</p>{/if}
			<button class="btn btn-primary w-full" disabled={busy || !ready}>{busy ? 'Memproses…' : 'Masuk'}</button>
		</form>
	</div>
</div>
