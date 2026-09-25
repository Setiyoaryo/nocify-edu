<script lang="ts">
	import { patch, post, del, ApiError } from '$lib/api';
	import { session } from '$lib/session.svelte';
	import { goto } from '$app/navigation';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';

	let name = $state(session.user?.display_name ?? '');
	let nameMsg = $state('');
	let pw = $state({ current_password: '', new_password: '' });
	let pwFields = $state<Record<string, string>>({});
	let pwMsg = $state('');
	let delPw = $state('');
	let delErr = $state('');

	async function saveName(e: SubmitEvent) {
		e.preventDefault();
		try {
			session.user = (await patch('/me', { display_name: name })).user;
			nameMsg = 'Tersimpan';
		} catch (err) {
			nameMsg = err instanceof ApiError ? (err.fields.display_name ?? err.message) : 'Gagal.';
		}
	}
	async function savePw(e: SubmitEvent) {
		e.preventDefault();
		pwFields = {};
		pwMsg = '';
		try {
			await post('/me/password', pw);
			pw = { current_password: '', new_password: '' };
			pwMsg = 'Kata sandi diganti. Sesi di perangkat lain sudah dikeluarkan.';
		} catch (err) {
			if (err instanceof ApiError) pwFields = err.fields;
		}
	}
	async function deleteAccount(e: SubmitEvent) {
		e.preventDefault();
		delErr = '';
		if (!confirm('Hapus akun, semua profil anak, dan seluruh riwayat belajar secara permanen?')) return;
		try {
			await del('/me', { password: delPw });
			session.user = null;
			goto('/');
		} catch (err) {
			delErr = err instanceof ApiError ? (err.fields.password ?? err.message) : 'Gagal.';
		}
	}
</script>

<svelte:head><title>Pengaturan | EduPlay</title></svelte:head>

<h1 class="text-3xl sm:text-4xl">Pengaturan</h1>

<div class="mt-6 grid max-w-3xl gap-6">
	<form onsubmit={saveName} class="card space-y-4 p-6 sm:p-8">
		<h2 class="text-xl">Profil</h2>
		<p class="text-sm text-ink-soft">Email: <b class="text-ink">{session.user?.email}</b></p>
		<div><label class="label" for="dn">Nama</label><input id="dn" class="field" bind:value={name} /></div>
		<div class="flex items-center gap-3"><button class="btn btn-primary">Simpan nama</button>{#if nameMsg}<span class="flex items-center gap-1.5 font-bold text-leaf-deep" role="status"><CheckCircle size={18} weight="fill" aria-hidden="true" />{nameMsg}</span>{/if}</div>
	</form>

	<form onsubmit={savePw} class="card space-y-4 p-6 sm:p-8">
		<h2 class="text-xl">Kata sandi</h2>
		<div>
			<label class="label" for="cp">Kata sandi saat ini</label>
			<input id="cp" class="field" type="password" autocomplete="current-password" bind:value={pw.current_password} />
			{#if pwFields.current_password}<p class="field-error">{pwFields.current_password}</p>{/if}
		</div>
		<div>
			<label class="label" for="np">Kata sandi baru</label>
			<input id="np" class="field" type="password" autocomplete="new-password" bind:value={pw.new_password} />
			{#if pwFields.new_password}<p class="field-error">{pwFields.new_password}</p>{/if}
		</div>
		<button class="btn btn-primary">Ganti kata sandi</button>
		{#if pwMsg}<p class="font-semibold text-leaf-deep" role="status">{pwMsg}</p>{/if}
	</form>

	<form onsubmit={deleteAccount} class="card space-y-4 border-2 border-error/20 p-6 sm:p-8">
		<h2 class="text-xl text-error">Hapus akun</h2>
		<p class="text-sm text-ink-soft">Menghapus akun Anda, semua profil anak, dan seluruh data belajarnya secara permanen. Langganan aktif ikut berhenti.</p>
		<div><label class="label" for="dp">Kata sandi</label><input id="dp" class="field" type="password" autocomplete="current-password" bind:value={delPw} /></div>
		{#if delErr}<p class="field-error" role="alert">{delErr}</p>{/if}
		<button class="btn btn-danger">Hapus akun permanen</button>
	</form>
</div>
