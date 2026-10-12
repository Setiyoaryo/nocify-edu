<script lang="ts">
 import { onMount } from 'svelte';
 import { get, post, ApiError } from '$lib/api';
 import { goto } from '$app/navigation';
 let displayName = $state('');
 let email = $state('');
 let password = $state('');
 let consent = $state(false);
 let loading = $state(true);
 let busy = $state(false);
 let error = $state('');
 onMount(async () => {
  try {
   const p = await get<{ email: string; display_name: string }>('/auth/google/pending');
   email = p.email; displayName = p.display_name;
  } catch { error = 'Sesi Google berakhir. Silakan masuk kembali.'; }
  finally { loading = false; }
 });
 async function submit(e: SubmitEvent) {
  e.preventDefault(); busy = true; error = '';
  try {
   await post('/auth/google/complete', { display_name: displayName, password, consent });
   await goto('/app/parent/children/new?welcome=1');
  } catch (err) { error = err instanceof ApiError ? err.message : 'Pendaftaran gagal.'; }
  finally { busy = false; }
 }
</script>
<svelte:head><title>Daftar Google | Nocify EDU</title><meta name="robots" content="noindex,nofollow" /></svelte:head>
<main class="mx-auto flex min-h-screen max-w-lg items-center px-5 py-12">
 <section class="card w-full space-y-5 p-6 sm:p-8">
  <h1 class="text-2xl font-bold">Selesaikan akun keluarga</h1>
  <p class="text-sm text-ink-soft">Google sudah memverifikasi identitas Anda. Buat kata sandi pemulihan dan setujui penggunaan data sebelum menambahkan profil anak.</p>
  {#if loading}<p>Memeriksa sesi…</p>
  {:else if email}
  <form class="space-y-4" onsubmit={submit}>
   <div><label class="label" for="ge">Email Google</label><input id="ge" class="field" value={email} readonly /></div>
   <div><label class="label" for="gn">Nama orang tua</label><input id="gn" class="field" bind:value={displayName} maxlength="60" required /></div>
   <div><label class="label" for="gp">Kata sandi pemulihan (minimal 8 karakter)</label><input id="gp" class="field" type="password" bind:value={password} minlength="8" maxlength="72" autocomplete="new-password" required /></div>
   <label class="flex items-start gap-3 text-sm"><input type="checkbox" class="mt-1" bind:checked={consent} required /><span>Saya orang tua/wali dan menyetujui pengolahan data anak sesuai <a href="/privacy" target="_blank" class="underline">Kebijakan Privasi</a> dan <a href="/terms" target="_blank" class="underline">Syarat Layanan</a>.</span></label>
   {#if error}<p class="field-error" role="alert">{error}</p>{/if}
   <button class="btn btn-primary w-full" disabled={busy || !consent}>{busy ? 'Menyimpan…' : 'Buat akun keluarga'}</button>
  </form>
  {:else}<p class="field-error">{error}</p><a class="btn btn-primary" href="/login?role=parent">Kembali login</a>{/if}
 </section>
</main>
