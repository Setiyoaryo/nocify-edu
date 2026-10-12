<script lang="ts">
	import Meta from '$lib/components/Meta.svelte';
	import AuthShell from '$lib/components/AuthShell.svelte';
	import { post, ApiError } from '$lib/api';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount, onDestroy } from 'svelte';
	import art from '$lib/illustrations/parent-child.svg';
	import WhatsappLogo from 'phosphor-svelte/lib/WhatsappLogo';
	import LockKey from 'phosphor-svelte/lib/LockKey';
	import ArrowLeft from 'phosphor-svelte/lib/ArrowLeft';
	import WarningCircle from 'phosphor-svelte/lib/WarningCircle';
	import EnvelopeSimple from 'phosphor-svelte/lib/EnvelopeSimple';

	let ready = $state(false);
	onMount(() => {
		ready = true;
		if (page.url.searchParams.get('email') === '1' || page.url.searchParams.get('email') === 'true') {
			showEmailFallback = true;
		}
	});

	type Step = 'form' | 'otp';
	let step = $state<Step>('form');

	let form = $state({ display_name: '', phone: '', consent: false });
	let emailForm = $state({ email: '', password: '' });
	let showEmailFallback = $state(false);

	let fields = $state<Record<string, string>>({});
	let formattedPhone = $state('');

	// OTP form (6 digits)
	let otpDigits = $state<string[]>(['', '', '', '', '', '']);
	let countdown = $state(0);
	let timer: any = null;

	let error = $state('');
	let busy = $state(false);

	onDestroy(() => {
		if (timer) clearInterval(timer);
	});

	function startTimer(seconds = 60) {
		if (timer) clearInterval(timer);
		countdown = seconds;
		timer = setInterval(() => {
			if (countdown > 0) {
				countdown -= 1;
			} else {
				clearInterval(timer);
				timer = null;
			}
		}, 1000);
	}

	async function submitSendOTP(e?: SubmitEvent) {
		if (e) e.preventDefault();
		error = '';
		fields = {};

		const cleanName = form.display_name.trim();
		const cleanPhone = form.phone.trim();

		if (!cleanName) {
			fields.display_name = 'Nama lengkap atau panggilan wajib diisi.';
		}
		if (!cleanPhone) {
			fields.phone = 'Nomor WhatsApp wajib diisi.';
		}
		if (!form.consent) {
			fields.consent = 'Persetujuan orang tua/wali diperlukan untuk mendaftar.';
		}

		if (Object.keys(fields).length > 0) {
			return;
		}

		busy = true;
		try {
			const res = await post<{ status: string; phone: string; message: string }>('/auth/otp/send', {
				phone: cleanPhone,
				role: 'parent'
			});
			formattedPhone = res.phone;
			step = 'otp';
			startTimer(60);
			queueMicrotask(() => {
				document.getElementById('otp-0')?.focus();
			});
		} catch (err) {
			if (err instanceof ApiError) {
				fields = err.fields ?? {};
				if (!Object.keys(fields).length) {
					error = err.message;
				}
			} else {
				error = 'Gagal mengirim kode OTP. Periksa nomor WhatsApp Anda dan coba lagi.';
			}
		} finally {
			busy = false;
		}
	}

	function handleDigitInput(index: number, e: Event) {
		const target = e.target as HTMLInputElement;
		const val = target.value.replace(/\D/g, '');
		error = '';

		if (val.length > 1) {
			const digits = val.slice(0, 6).split('');
			for (let i = 0; i < 6; i++) {
				if (i >= index && i < index + digits.length) {
					otpDigits[i] = digits[i - index] ?? '';
				}
			}
			target.value = otpDigits[index] ?? '';
			const nextIdx = Math.min(index + digits.length - 1, 5);
			const nextEl = document.getElementById(`otp-${nextIdx}`) as HTMLInputElement | null;
			nextEl?.focus();
			return;
		}

		otpDigits[index] = val ? val[val.length - 1] : '';

		if (val && index < 5) {
			const nextInput = document.getElementById(`otp-${index + 1}`) as HTMLInputElement | null;
			nextInput?.focus();
		}
	}

	function handleDigitKeydown(index: number, e: KeyboardEvent) {
		if (e.key === 'Backspace' && !otpDigits[index] && index > 0) {
			const prevInput = document.getElementById(`otp-${index - 1}`) as HTMLInputElement | null;
			prevInput?.focus();
		} else if (e.key === 'ArrowLeft' && index > 0) {
			const prevInput = document.getElementById(`otp-${index - 1}`) as HTMLInputElement | null;
			prevInput?.focus();
		} else if (e.key === 'ArrowRight' && index < 5) {
			const nextInput = document.getElementById(`otp-${index + 1}`) as HTMLInputElement | null;
			nextInput?.focus();
		}
	}

	function handlePaste(e: ClipboardEvent) {
		e.preventDefault();
		const pasted = e.clipboardData?.getData('text') ?? '';
		const digits = pasted.replace(/\D/g, '').slice(0, 6).split('');
		if (digits.length) {
			for (let i = 0; i < 6; i++) {
				otpDigits[i] = digits[i] ?? '';
			}
			const lastIdx = Math.min(digits.length, 5);
			(document.getElementById(`otp-${lastIdx}`) as HTMLInputElement)?.focus();
		}
	}

	async function resendOTP() {
		if (countdown > 0 || busy) return;
		await submitSendOTP();
	}

	async function submitVerifyOTP(e?: SubmitEvent) {
		if (e) e.preventDefault();
		const code = otpDigits.join('');
		if (code.length < 6) {
			error = 'Masukkan 6 digit kode OTP secara lengkap.';
			return;
		}

		busy = true;
		error = '';
		try {
			await post('/auth/otp/verify', {
				phone: formattedPhone || form.phone.trim(),
				code,
				role: 'parent',
				display_name: form.display_name.trim()
			});
			await goto('/app/parent/children/new?welcome=1');
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Kode OTP tidak valid atau telah kadaluarsa.';
		} finally {
			busy = false;
		}
	}

	async function submitEmailRegister(e: SubmitEvent) {
		e.preventDefault();
		error = '';
		fields = {};

		const cleanName = form.display_name.trim();
		const cleanEmail = emailForm.email.trim();
		const password = emailForm.password;

		if (!cleanName) fields.display_name = 'Nama lengkap wajib diisi.';
		if (!cleanEmail) fields.email = 'Email wajib diisi.';
		if (!password) fields.password = 'Kata sandi minimal 8 karakter.';
		if (!form.consent) fields.consent = 'Persetujuan orang tua/wali diperlukan.';

		if (Object.keys(fields).length > 0) return;

		busy = true;
		try {
			await post('/auth/register', {
				display_name: cleanName,
				email: cleanEmail,
				password,
				consent: form.consent
			});
			await goto('/app/parent/children/new?welcome=1');
		} catch (err) {
			if (err instanceof ApiError) {
				fields = err.fields ?? {};
				if (!Object.keys(fields).length) error = err.message;
			} else {
				error = 'Gagal mendaftar dengan email.';
			}
		} finally {
			busy = false;
		}
	}
</script>

<Meta title="Daftar Akun Orang Tua" description="Buat akun orang tua EduPlay dengan nomor WhatsApp untuk mendampingi belajar anak." />

<AuthShell
	title="Buat akun orang tua"
	lead="Gratis untuk 1 anak. Profil anak dibuat setelah ini."
	image={art}
	points={[
		'Gratis untuk 1 anak, tanpa kartu kredit',
		'Challenge 5–10 menit sesuai kelas',
		'Laporan yang menjawab: sudah bisa apa, perlu latihan apa'
	]}
>
	{#if step === 'form'}
		<form class="mt-6 sm:mt-8 space-y-4 sm:space-y-5" method="post" onsubmit={submitSendOTP} novalidate>
			<div>
				<label class="label text-xs sm:text-sm" for="name">Nama Lengkap / Panggilan Anda</label>
				<input
					class="field text-sm sm:text-base font-semibold"
					id="name"
					autocomplete="name"
					placeholder="Contoh: Ayah Budi / Bunda Siti"
					bind:value={form.display_name}
					aria-invalid={!!fields.display_name}
					aria-describedby={fields.display_name ? 'name-err' : undefined}
				/>
				{#if fields.display_name}
					<p id="name-err" class="field-error mt-1 text-xs sm:text-sm">{fields.display_name}</p>
				{/if}
			</div>

			<div>
				<label class="label text-xs sm:text-sm" for="phone">Nomor WhatsApp Aktif</label>
				<div
					class="relative flex items-center rounded-2xl border border-line bg-paper-deep transition-all focus-within:border-primary focus-within:ring-2 focus-within:ring-primary/20"
				>
					<div class="flex items-center gap-2 border-r border-line/80 px-3 sm:px-3.5 py-3 text-sm font-bold text-ink select-none sm:text-base shrink-0">
						<svg class="h-3.5 w-5 rounded-[3px] border border-line/80 shadow-xs overflow-hidden shrink-0" viewBox="0 0 20 14" aria-hidden="true">
							<rect width="20" height="7" fill="#E11D48" />
							<rect y="7" width="20" height="7" fill="#FFFFFF" />
						</svg>
						<span>+62</span>
					</div>
					<input
						class="w-full bg-transparent px-3 sm:px-3.5 py-3 text-base font-semibold text-ink placeholder:text-ink-soft/60 focus:outline-none"
						id="phone"
						type="tel"
						inputmode="numeric"
						placeholder="812-3456-7890"
						autocomplete="tel-national"
						required
						bind:value={form.phone}
						aria-invalid={!!fields.phone}
						aria-describedby={fields.phone ? 'phone-err' : 'phone-hint'}
					/>
				</div>
				{#if fields.phone}
					<p id="phone-err" class="field-error mt-1 text-xs sm:text-sm">{fields.phone}</p>
				{:else}
					<p id="phone-hint" class="mt-1.5 text-xs text-ink-soft">
						Bisa diawali 08 atau langsung 8 (contoh: 081234567890). Kode OTP 6 digit akan dikirim ke WhatsApp ini.
					</p>
				{/if}
			</div>

			<div class="rounded-2xl border border-line bg-paper-deep p-4">
				<label class="flex cursor-pointer gap-3 text-xs sm:text-sm leading-relaxed text-ink">
					<input
						type="checkbox"
						class="mt-0.5 size-5 shrink-0 rounded-lg accent-primary"
						bind:checked={form.consent}
						aria-invalid={!!fields.consent}
					/>
					<span>
						Saya orang tua/wali anak yang akan menggunakan EduPlay dan menyetujui pemrosesan data anak untuk keperluan belajarnya sesuai <a href="/privacy" target="_blank" class="text-primary-deep font-bold underline">Kebijakan Privasi</a> dan <a href="/terms" target="_blank" class="text-primary-deep font-bold underline">Syarat Layanan</a>.
					</span>
				</label>
				{#if fields.consent}
					<p class="field-error mt-2 text-xs sm:text-sm">{fields.consent}</p>
				{/if}
			</div>

			{#if error && !showEmailFallback}
				<div class="rounded-2xl border border-red-200 bg-red-50 p-3.5 text-xs sm:text-sm text-red-800 shadow-sm" role="alert">
					<div class="flex items-start gap-2.5">
						<span class="grid size-6 place-items-center rounded-lg bg-red-100 text-red-700 shrink-0 mt-0.5">
							<WarningCircle size={18} weight="fill" aria-hidden="true" />
						</span>
						<p class="font-bold flex-1">{error}</p>
					</div>
				</div>
			{/if}

			<button
				type="submit"
				class="btn btn-primary w-full py-3.5 text-base font-bold shadow-md inline-flex items-center justify-center gap-2"
				disabled={busy || !ready}
			>
				{#if busy}
					<span class="anim-spin inline-block size-4 rounded-full border-2 border-white border-t-transparent" aria-hidden="true"></span>
					<span>Mengirim kode OTP…</span>
				{:else}
					<WhatsappLogo size={22} weight="fill" aria-hidden="true" />
					<span>Daftar dengan WhatsApp</span>
				{/if}
			</button>
		</form>

		<!-- Collapsible Email/Password Fallback for Staff/Legacy users -->
		<div class="mt-4 pt-3 text-center border-t border-line/60">
			<button
				type="button"
				class="inline-flex items-center gap-1.5 text-xs font-semibold text-ink-soft hover:text-ink transition-colors underline"
				onclick={() => (showEmailFallback = !showEmailFallback)}
			>
				<EnvelopeSimple size={14} weight="bold" aria-hidden="true" />
				<span>{showEmailFallback ? 'Sembunyikan daftar email' : 'Atau daftar dengan email & kata sandi'}</span>
			</button>

			{#if showEmailFallback}
				<form
					onsubmit={submitEmailRegister}
					novalidate
					class="mt-4 space-y-3.5 text-left rounded-2xl border border-line bg-paper-deep p-4 shadow-xs"
				>
					<div>
						<label class="label text-xs sm:text-sm" for="email">Email</label>
						<input
							class="field text-sm sm:text-base font-semibold"
							id="email"
							type="email"
							autocomplete="email"
							placeholder="nama@email.com"
							bind:value={emailForm.email}
							aria-invalid={!!fields.email}
						/>
						{#if fields.email}<p class="field-error mt-1 text-xs">{fields.email}</p>{/if}
					</div>

					<div>
						<label class="label text-xs sm:text-sm" for="password">Kata sandi</label>
						<input
							class="field text-sm sm:text-base font-semibold"
							id="password"
							type="password"
							autocomplete="new-password"
							placeholder="Minimal 8 karakter"
							bind:value={emailForm.password}
							aria-invalid={!!fields.password}
						/>
						{#if fields.password}<p class="field-error mt-1 text-xs">{fields.password}</p>{/if}
					</div>

					{#if error}
						<div class="rounded-xl border border-red-200 bg-red-50 p-2.5 text-xs text-red-800" role="alert">
							{error}
						</div>
					{/if}

					<button
						type="submit"
						class="btn btn-outline btn-sm w-full font-bold"
						disabled={busy || !ready}
					>
						{busy ? 'Memproses…' : 'Buat akun'}
					</button>
				</form>
			{/if}
		</div>

		<p class="mt-6 text-center text-xs sm:text-sm text-ink-soft">
			Sudah punya akun? <a class="font-bold text-primary-deep hover:underline" href="/login?role=parent">Masuk</a>
		</p>
	{:else}
		<!-- STEP 2: VERIFY OTP -->
		<div class="mt-6 sm:mt-8 space-y-5">
			<div>
				<h2 class="text-xl sm:text-2xl font-extrabold text-ink">Verifikasi Kode OTP</h2>
				<p class="mt-1 text-xs sm:text-sm text-ink-soft">
					Kode 6 digit telah dikirim melalui WhatsApp ke:
					<strong class="text-ink font-bold">{formattedPhone || form.phone}</strong>
				</p>
				<button
					type="button"
					class="mt-2 inline-flex items-center gap-1.5 text-xs font-bold text-primary-deep hover:underline"
					onclick={() => {
						step = 'form';
						error = '';
					}}
				>
					<ArrowLeft size={14} weight="bold" aria-hidden="true" />
					<span>Ubah nama atau nomor WhatsApp</span>
				</button>
			</div>

			<form onsubmit={submitVerifyOTP} novalidate class="space-y-5">
				<div>
					<label class="label block text-center text-xs sm:text-sm" for="otp-0">
						Masukkan 6 Digit Kode OTP WhatsApp
					</label>
					<div class="mt-3 flex justify-center gap-1.5 sm:gap-3">
						{#each otpDigits as _, i}
							<input
								id="otp-{i}"
								type="text"
								inputmode="numeric"
								pattern="[0-9]*"
								maxlength={i === 0 ? 6 : 1}
								class="h-11 sm:h-13 w-full max-w-[48px] min-w-0 flex-1 text-center text-lg sm:text-2xl font-black rounded-xl sm:rounded-2xl border-2 border-line bg-paper-deep text-ink transition-all focus:border-primary focus:bg-white focus:outline-none focus:ring-2 focus:ring-primary/20"
								bind:value={otpDigits[i]}
								oninput={(e) => handleDigitInput(i, e)}
								onkeydown={(e) => handleDigitKeydown(i, e)}
								onpaste={i === 0 ? handlePaste : undefined}
								autocomplete={i === 0 ? 'one-time-code' : 'off'}
							/>
						{/each}
					</div>
				</div>

				<div class="flex items-center justify-between text-xs sm:text-sm">
					<span class="text-ink-soft">Tidak menerima kode?</span>
					{#if countdown > 0}
						<span class="font-bold text-ink-soft">Kirim ulang ({countdown}s)</span>
					{:else}
						<button
							type="button"
							class="font-bold text-primary-deep hover:underline disabled:opacity-50"
							disabled={busy}
							onclick={resendOTP}
						>
							Kirim Ulang Kode
						</button>
					{/if}
				</div>

				{#if error}
					<div class="rounded-2xl border border-red-200 bg-red-50 p-3 text-xs sm:text-sm text-red-800 text-center font-bold" role="alert">
						{error}
					</div>
				{/if}

				<button
					type="submit"
					class="btn btn-primary w-full py-3.5 text-base font-bold shadow-md inline-flex items-center justify-center gap-2"
					disabled={busy || otpDigits.join('').length < 6}
				>
					{#if busy}
						<span class="anim-spin inline-block size-4 rounded-full border-2 border-white border-t-transparent" aria-hidden="true"></span>
						<span>Memverifikasi…</span>
					{:else}
						<LockKey size={20} weight="bold" aria-hidden="true" />
						<span>Verifikasi & Buat Akun</span>
					{/if}
				</button>
			</form>

			<p class="pt-2 text-center text-xs text-ink-soft">
				Pastikan akun WhatsApp aktif dan dapat menerima pesan dari EduPlay.
			</p>
		</div>
	{/if}
</AuthShell>
