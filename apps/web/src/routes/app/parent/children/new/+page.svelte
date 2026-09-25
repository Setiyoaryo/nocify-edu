<script lang="ts">
	import { post, ApiError } from '$lib/api';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import ChildForm from '../ChildForm.svelte';
	import friends from '$lib/illustrations/friends.svg';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';

	let value = $state({ nickname: '', grade_id: null as number | null, avatar: 'cat', pin: '' });
	let fields = $state<Record<string, string>>({});
	let error = $state('');
	let busy = $state(false);
	const welcome = page.url.searchParams.has('welcome');

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		fields = {};
		error = '';
		try {
			const { child } = await post('/children', { ...value, pin: value.pin || undefined });
			await goto(`/app/play/${child.id}`);
		} catch (err) {
			if (err instanceof ApiError) {
				fields = err.fields;
				if (!Object.keys(err.fields).length) error = err.message;
			}
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Tambah anak | EduPlay</title></svelte:head>

<div class="grid gap-8 lg:grid-cols-[1fr_20rem]">
	<div class="card p-6 sm:p-8">
		{#if welcome}<p class="pill mb-3 bg-leaf-tint text-leaf-deep"><CheckCircle size={16} weight="fill" aria-hidden="true" />Akun Anda sudah jadi</p>{/if}
		<h1 class="text-3xl">{welcome ? 'Sekarang, profil anak' : 'Tambah profil anak'}</h1>
		<p class="mt-2 text-ink-soft">Satu profil untuk setiap anak, supaya laporan dan soal sesuai kelasnya.</p>
		<form class="mt-8" onsubmit={submit} novalidate>
			<ChildForm bind:value {fields} />
			{#if error}<p class="field-error mt-4" role="alert">{error} {#if error.includes('Berlangganan')}<a class="underline" href="/app/parent/subscription">Lihat paket</a>{/if}</p>{/if}
			<button class="btn btn-primary mt-8 w-full sm:w-auto" disabled={busy}>{busy ? 'Menyimpan…' : 'Simpan dan mulai bermain'}</button>
		</form>
	</div>
	<img src={friends} alt="" class="mx-auto hidden w-full max-w-xs self-center lg:block" />
</div>
