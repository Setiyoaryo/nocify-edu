<script lang="ts">
	import Meta from '$lib/components/Meta.svelte';
	import AuthShell from '$lib/components/AuthShell.svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import { post, ApiError } from '$lib/api';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount, onDestroy } from 'svelte';
	import loginArt from '$lib/illustrations/login.svg';
	import playArt from '$lib/illustrations/play.svg';
	import type { Child, User } from '$lib/types';
	import WhatsappLogo from 'phosphor-svelte/lib/WhatsappLogo';
	import UsersThree from 'phosphor-svelte/lib/UsersThree';
	import GameController from 'phosphor-svelte/lib/GameController';
	import Sparkle from 'phosphor-svelte/lib/Sparkle';
	import ArrowLeft from 'phosphor-svelte/lib/ArrowLeft';
	import ArrowRight from 'phosphor-svelte/lib/ArrowRight';
	import Rocket from 'phosphor-svelte/lib/Rocket';
	import LockKey from 'phosphor-svelte/lib/LockKey';
	import EnvelopeSimple from 'phosphor-svelte/lib/EnvelopeSimple';

	let ready = $state(false);
	onMount(() => (ready = true));

	type RoleChoice = 'parent' | 'child' | null;
	type Step = 'phone' | 'otp' | 'pick_child';

	let role = $state<RoleChoice>(null);
	let step = $state<Step>('phone');

	// Phone form
	let phone = $state('');
	let formattedPhone = $state('');

	// OTP form (6 digits)
	let otpDigits = $state<string[]>(['', '', '', '', '', '']);
	let devCode = $state('');
	let countdown = $state(0);
	let timer: any = null;

	// Child selection state
	let verifiedChildren = $state<Child[]>([]);

	// Email/Password fallback (for parent/staff)
	let showEmailFallback = $state(false);
	let email = $state('');
	let password = $state('');

	let error = $state('');
	let busy = $state(false);

	onMount(() => {
		const paramRole = page.url.searchParams.get('role');
		if (paramRole === 'parent' || paramRole === 'child') {
			role = paramRole;
		}
	});

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

	function selectRole(r: 'parent' | 'child') {
		role = r;
		step = 'phone';
		error = '';
	}

	function resetRole() {
		role = null;
		step = 'phone';
		error = '';
		otpDigits = ['', '', '', '', '', ''];
	}

	async function submitSendOTP(e?: SubmitEvent) {
		if (e) e.preventDefault();
		const cleanPhone = phone.trim();
		if (!cleanPhone) {
			error = role === 'child' ? 'Masukkan nomor WhatsApp Ayah atau Bunda ya!' : 'Masukkan nomor WhatsApp Anda.';
			return;
		}

		busy = true;
		error = '';
		try {
			const res = await post<{ status: string; phone: string; message: string; dev_code?: string }>('/auth/otp/send', {
				phone: cleanPhone,
				role: role ?? 'parent'
			});
			formattedPhone = res.phone;
			devCode = res.dev_code ?? '';
			step = 'otp';
			startTimer(60);
			// Auto focus first OTP input box after DOM updates
			setTimeout(() => {
				document.getElementById('otp-0')?.focus();
			}, 60);
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Gagal mengirim kode OTP. Coba beberapa saat lagi.';
		} finally {
			busy = false;
		}
	}

	function handleDigitInput(index: number, e: Event) {
		const target = e.target as HTMLInputElement;
		const val = target.value.replace(/\D/g, '');
		otpDigits[index] = val ? val[val.length - 1] : '';
		error = '';

		if (val && index < 5) {
			const nextInput = document.getElementById(`otp-${index + 1}`) as HTMLInputElement | null;
			nextInput?.focus();
		}
	}

	function handleDigitKeydown(index: number, e: KeyboardEvent) {
		if (e.key === 'Backspace' && !otpDigits[index] && index > 0) {
			const prevInput = document.getElementById(`otp-${index - 1}`) as HTMLInputElement | null;
			prevInput?.focus();
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

	function fillDevCode() {
		if (!devCode) return;
		const chars = devCode.slice(0, 6).split('');
		for (let i = 0; i < 6; i++) {
			otpDigits[i] = chars[i] ?? '';
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
			const res = await post<{
				role: string;
				user: User;
				children?: Child[];
			}>('/auth/otp/verify', {
				phone: formattedPhone || phone,
				code,
				role: role ?? 'parent'
			});

			if (role === 'child') {
				const children = res.children ?? [];
				if (children.length === 1) {
					await goto(`/app/play/${children[0].id}`);
					return;
				} else if (children.length > 1) {
					verifiedChildren = children;
					step = 'pick_child';
					return;
				} else {
					verifiedChildren = [];
					step = 'pick_child';
					return;
				}
			} else {
				const next = new URLSearchParams(location.search).get('next');
				await goto(next?.startsWith('/app') ? next : res.user.role === 'parent' ? '/app/parent' : '/app/admin');
			}
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Kode OTP tidak valid atau telah kadaluarsa.';
		} finally {
			busy = false;
		}
	}

	async function chooseChild(childId: string) {
		await goto(`/app/play/${childId}`);
	}

	// Legacy / Staff Email Login
	async function submitEmailLogin(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			const { user } = await post<{ user: User }>('/auth/login', { email, password });
			const next = new URLSearchParams(location.search).get('next');
			await goto(next?.startsWith('/app') ? next : user.role === 'parent' ? '/app/parent' : '/app/admin');
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Gagal masuk. Periksa email dan kata sandi.';
		} finally {
			busy = false;
		}
	}
</script>

<Meta
	title={role === 'child' ? 'Masuk Ruang Anak' : 'Masuk'}
	description="Masuk ke EduPlay dengan WhatsApp OTP untuk orang tua atau anak."
/>

<AuthShell
	title={role === 'child' ? 'Dunia Petualangan EduPlay' : 'Selamat datang kembali'}
	lead={role === 'child'
		? 'Ayo sambung petualanganmu, selesaikan tantangan seru, dan kumpulkan bintang hari ini!'
		: 'Masuk untuk melihat perkembangan anak dan melanjutkan belajar.'}
	image={role === 'child' ? playArt : loginArt}
	points={role === 'child'
		? [
				'Kuis interaktif yang seru & penuh warna',
				'Kumpulkan bintang dan buka hadiah karakter baru',
				'Belajar mandiri aman tanpa iklan yang mengganggu'
			]
		: [
				'Laporan per skill untuk setiap anak',
				'Misi latihan yang bisa Anda tentukan',
				'Aman: tanpa iklan untuk anak'
			]}
>
	<!-- STAGE 1: ROLE SELECTION -->
	{#if role === null}
		<div class="mt-4">
			<h2 class="text-xl font-bold text-ink sm:text-2xl">Masuk sebagai siapa?</h2>
			<p class="mt-1 text-sm text-ink-soft">Pilih peran Anda untuk pengalaman belajar yang tepat.</p>

			<div class="mt-6 grid gap-4 sm:grid-cols-2">
				<!-- Orang Tua Card -->
				<button
					type="button"
					class="press group relative flex flex-col items-start gap-4 rounded-3xl border-2 border-line bg-white p-5 text-left transition-all hover:border-primary hover:bg-primary-tint/20 hover:shadow-md active:scale-[0.98]"
					onclick={() => selectRole('parent')}
				>
					<div class="flex w-full items-center justify-between">
						<span
							class="grid size-13 place-items-center rounded-2xl bg-primary-tint text-primary-deep shadow-sm transition-transform group-hover:scale-105"
						>
							<UsersThree size={30} weight="fill" aria-hidden="true" />
						</span>
						<span class="pill bg-primary/10 px-2.5 py-1 text-xs font-bold text-primary-deep">Area Orang Tua</span>
					</div>
					<div>
						<h3 class="text-lg font-extrabold text-ink sm:text-xl">Orang Tua</h3>
						<p class="mt-1.5 text-xs sm:text-sm leading-relaxed text-ink-soft">
							Pantau laporan skill, buat misi mingguan, dan kelola akun anak.
						</p>
					</div>
					<div class="mt-auto flex w-full items-center justify-between pt-2 text-xs sm:text-sm font-bold text-primary-deep">
						<span>Masuk Orang Tua</span>
						<ArrowRight size={16} weight="bold" class="transition-transform group-hover:translate-x-1" aria-hidden="true" />
					</div>
				</button>

				<!-- Anak Card (Kid Friendly) -->
				<button
					type="button"
					class="press group relative flex flex-col items-start gap-4 rounded-3xl border-2 border-sun/60 bg-gradient-to-b from-white to-sun-tint/30 p-5 text-left transition-all hover:border-sun hover:bg-sun-tint/40 hover:shadow-md active:scale-[0.98]"
					onclick={() => selectRole('child')}
				>
					<div class="flex w-full items-center justify-between">
						<span
							class="grid size-13 place-items-center rounded-2xl bg-sun-tint text-sun-deep shadow-sm transition-transform group-hover:scale-105"
						>
							<GameController size={30} weight="fill" aria-hidden="true" />
						</span>
						<span class="pill bg-sun/20 px-2.5 py-1 text-xs font-black tracking-wide text-sun-deep">Ruang Bermain ⭐</span>
					</div>
					<div>
						<h3 class="text-lg font-black text-ink sm:text-xl">Anak</h3>
						<p class="mt-1.5 text-xs sm:text-sm leading-relaxed text-ink-soft">
							Mulai petualangan kuis seru, raih bintang, dan koleksi avatar lucu!
						</p>
					</div>
					<div class="mt-auto flex w-full items-center justify-between pt-2 text-xs sm:text-sm font-black text-sun-deep">
						<span>Ayo Main Sekarang!</span>
						<Rocket size={16} weight="fill" class="transition-transform group-hover:translate-x-1" aria-hidden="true" />
					</div>
				</button>
			</div>

			<p class="mt-8 text-center text-xs sm:text-sm text-ink-soft">
				Belum punya akun? <a class="font-bold text-primary-deep hover:underline" href="/register">Daftar gratis</a>
			</p>
		</div>

	<!-- STAGE 2: FORM FOR SELECTED ROLE -->
	{:else}
		<!-- Header with role switch and back button -->
		<div class="mt-2 mb-6 flex flex-wrap items-center justify-between gap-3 border-b border-line pb-4">
			<button
				type="button"
				class="inline-flex items-center gap-1.5 text-xs sm:text-sm font-bold text-ink-soft hover:text-ink transition-colors"
				onclick={resetRole}
			>
				<ArrowLeft size={16} weight="bold" aria-hidden="true" />
				<span>Ganti peran</span>
			</button>

			<div class="flex items-center rounded-2xl bg-paper-deep p-1 shadow-inner" role="tablist">
				<button
					type="button"
					role="tab"
					aria-selected={role === 'parent'}
					class="flex items-center gap-1.5 rounded-xl px-3 py-1.5 text-xs sm:text-sm font-bold transition-all {role === 'parent'
						? 'bg-white text-primary-deep shadow-sm'
						: 'text-ink-soft hover:text-ink'}"
					onclick={() => selectRole('parent')}
				>
					<UsersThree size={16} weight="fill" aria-hidden="true" />
					<span>Orang Tua</span>
				</button>
				<button
					type="button"
					role="tab"
					aria-selected={role === 'child'}
					class="flex items-center gap-1.5 rounded-xl px-3 py-1.5 text-xs sm:text-sm font-bold transition-all {role === 'child'
						? 'bg-sun text-ink shadow-sm'
						: 'text-ink-soft hover:text-ink'}"
					onclick={() => selectRole('child')}
				>
					<GameController size={16} weight="fill" aria-hidden="true" />
					<span>Anak ⭐</span>
				</button>
			</div>
		</div>

		<!-- STEP 1: PHONE INPUT -->
		{#if step === 'phone'}
			<div class="space-y-5">
				<div>
					<h2 class="text-xl sm:text-2xl font-extrabold text-ink flex items-center gap-2">
						{#if role === 'child'}
							Hai Petualang Cilik! <span class="anim-float">🎒✨</span>
						{:else}
							Masuk sebagai Orang Tua
						{/if}
					</h2>
					<p class="mt-1 text-xs sm:text-sm text-ink-soft">
						{#if role === 'child'}
							Minta tolong Ayah atau Bunda ya untuk masukkan nomor WhatsApp mereka!
						{:else}
							Masukkan nomor WhatsApp aktif untuk menerima 6 digit kode OTP verifikasi.
						{/if}
					</p>
				</div>

				{#if role === 'child'}
					<div class="flex items-start gap-3 rounded-2xl border border-sun/40 bg-sun-tint/40 p-3.5 text-xs sm:text-sm text-ink shadow-sm">
						<span class="text-xl">💬</span>
						<p class="leading-relaxed font-semibold">
							Kode rahasia 6 angka akan dikirim ke WhatsApp Ayah atau Bunda. Nanti tinggal ketik kodenya untuk langsung bermain!
						</p>
					</div>
				{/if}

				<form onsubmit={submitSendOTP} novalidate class="space-y-5">
					<div>
						<label class="label text-xs sm:text-sm" for="phone">
							{role === 'child' ? 'Nomor WhatsApp Ayah / Bunda' : 'Nomor WhatsApp'}
						</label>
						<div
							class="relative flex items-center rounded-2xl border border-line bg-paper-deep transition-all focus-within:border-primary focus-within:ring-2 focus-within:ring-primary/20"
						>
							<div class="flex items-center gap-1.5 border-r border-line/80 px-3.5 py-3 text-sm font-bold text-ink select-none sm:text-base">
								<span>🇮🇩</span>
								<span>+62</span>
							</div>
							<input
								class="w-full bg-transparent px-3.5 py-3 text-sm sm:text-base font-semibold text-ink placeholder:text-ink-soft/60 focus:outline-none"
								id="phone"
								type="tel"
								inputmode="numeric"
								placeholder="812-3456-7890"
								autocomplete="tel-national"
								required
								bind:value={phone}
							/>
						</div>
						<p class="mt-1.5 text-xs text-ink-soft">
							{role === 'child' ? 'Bisa ketik 08... atau 8...' : 'Bisa diawali 08 atau langsung 8 (contoh: 081234567890)'}
						</p>
					</div>

					{#if error}<p class="field-error" role="alert">{error}</p>{/if}

					<button
						type="submit"
						class="press flex min-h-12 sm:min-h-13 w-full items-center justify-center gap-2 rounded-2xl py-3 px-5 text-sm sm:text-base font-bold shadow-md transition-all active:scale-[0.99] {role ===
						'child'
							? 'bg-sun hover:bg-sun-deep text-ink text-base sm:text-lg font-black'
							: 'bg-[#25D366] hover:bg-[#1eb956] text-white'}"
						disabled={busy || !ready}
					>
						{#if busy}
							<span>Mengirim kode…</span>
						{:else if role === 'child'}
							<Rocket size={22} weight="fill" aria-hidden="true" />
							<span>Kirim Kode ke Ayah / Bunda 🚀</span>
						{:else}
							<WhatsappLogo size={22} weight="fill" aria-hidden="true" />
							<span>Kirim Kode OTP via WhatsApp</span>
						{/if}
					</button>
				</form>

				<!-- Alternative Email Login for Parent/Staff -->
				{#if role === 'parent'}
					<div class="mt-6 border-t border-line pt-4">
						<button
							type="button"
							class="mx-auto flex items-center gap-1.5 text-xs font-bold text-ink-soft hover:text-ink transition-colors"
							onclick={() => (showEmailFallback = !showEmailFallback)}
						>
							<EnvelopeSimple size={15} weight="bold" />
							<span>{showEmailFallback ? 'Sembunyikan masuk email' : 'Atau masuk dengan email & kata sandi'}</span>
						</button>

						{#if showEmailFallback}
							<form class="mt-4 space-y-4 rounded-2xl bg-paper-deep p-4" method="post" onsubmit={submitEmailLogin} novalidate>
								<div>
									<label class="label text-xs" for="email">Email</label>
									<input class="field text-sm" id="email" type="email" autocomplete="email" required bind:value={email} />
								</div>
								<div>
									<label class="label text-xs" for="password">Kata sandi</label>
									<input
										class="field text-sm"
										id="password"
										type="password"
										autocomplete="current-password"
										required
										bind:value={password}
									/>
								</div>
								{#if error}<p class="field-error" role="alert">{error}</p>{/if}
								<button class="btn btn-primary w-full text-sm" disabled={busy || !ready}>
									{busy ? 'Memproses…' : 'Masuk dengan Email'}
								</button>
							</form>
						{/if}
					</div>
				{/if}

				<p class="mt-6 text-center text-xs sm:text-sm text-ink-soft">
					Belum punya akun? <a class="font-bold text-primary-deep hover:underline" href="/register">Daftar gratis</a>
				</p>
			</div>

		<!-- STEP 2: OTP VERIFICATION -->
		{:else if step === 'otp'}
			<div class="space-y-5">
				<div>
					<h2 class="text-xl sm:text-2xl font-extrabold text-ink">
						{role === 'child' ? 'Ketik Kode Rahasia 🔑' : 'Verifikasi Kode OTP'}
					</h2>
					<p class="mt-1 text-xs sm:text-sm text-ink-soft">
						Kode 6 digit telah dikirim melalui WhatsApp ke:
					</p>
					<div class="mt-2 flex items-center gap-2 font-mono text-sm font-bold text-ink sm:text-base">
						<span class="rounded-xl bg-paper-deep px-3 py-1">{formattedPhone}</span>
						<button
							type="button"
							class="text-xs font-bold text-primary-deep hover:underline"
							onclick={() => {
								step = 'phone';
								error = '';
							}}
						>
							Ubah nomor
						</button>
					</div>
				</div>

				<!-- Helper chip for dev/mock OTP -->
				{#if devCode}
					<button
						type="button"
						class="flex w-full items-center justify-between rounded-2xl border border-sun/50 bg-sun-tint/50 p-3 text-xs sm:text-sm text-ink transition-colors hover:bg-sun-tint"
						onclick={fillDevCode}
					>
						<span class="flex items-center gap-1.5 font-semibold">
							<Sparkle size={16} weight="fill" class="text-sun-deep" />
							<span>Kode OTP Uji Coba: <b class="font-mono text-base text-ink">{devCode}</b></span>
						</span>
						<span class="font-bold text-sun-deep underline">Isi otomatis</span>
					</button>
				{/if}

				<form onsubmit={submitVerifyOTP} novalidate class="space-y-5">
					<div>
						<label class="label block text-center text-xs sm:text-sm" for="otp-0">
							Masukkan 6 Digit Kode
						</label>
						<div class="mt-3 flex justify-center gap-2 sm:gap-3">
							{#each otpDigits as _, i}
								<input
									id="otp-{i}"
									type="text"
									inputmode="numeric"
									pattern="[0-9]*"
									maxlength="1"
									class="size-11 sm:size-13 text-center text-xl sm:text-2xl font-black rounded-2xl border-2 border-line bg-paper-deep text-ink transition-all focus:border-primary focus:bg-white focus:outline-none focus:ring-2 focus:ring-primary/20"
									bind:value={otpDigits[i]}
									oninput={(e) => handleDigitInput(i, e)}
									onkeydown={(e) => handleDigitKeydown(i, e)}
									onpaste={handlePaste}
									autocomplete={i === 0 ? 'one-time-code' : 'off'}
								/>
							{/each}
						</div>
					</div>

					{#if error}<p class="field-error text-center" role="alert">{error}</p>{/if}

					<button
						type="submit"
						class="press flex min-h-12 sm:min-h-13 w-full items-center justify-center gap-2 rounded-2xl py-3 px-5 text-sm sm:text-base font-bold shadow-md transition-all active:scale-[0.99] {role ===
						'child'
							? 'bg-sun hover:bg-sun-deep text-ink text-base sm:text-lg font-black'
							: 'btn-primary'}"
						disabled={busy || !ready || otpDigits.join('').length < 6}
					>
						{#if busy}
							<span>Memverifikasi…</span>
						{:else if role === 'child'}
							<Sparkle size={20} weight="fill" aria-hidden="true" />
							<span>Ayo Masuk & Main! ⭐</span>
						{:else}
							<LockKey size={20} weight="bold" aria-hidden="true" />
							<span>Verifikasi & Masuk</span>
						{/if}
					</button>

					<div class="text-center text-xs sm:text-sm text-ink-soft">
						{#if countdown > 0}
							<p>Kirim ulang kode dalam <span class="font-bold text-ink">{countdown} detik</span></p>
						{:else}
							<p>
								Tidak menerima kode?
								<button
									type="button"
									class="ml-1 font-bold text-primary-deep hover:underline"
									onclick={resendOTP}
									disabled={busy}
								>
									Kirim ulang kode OTP
								</button>
							</p>
						{/if}
					</div>
				</form>
			</div>

		<!-- STEP 3: CHILD PICKER (Anak Mode with Multiple Profiles) -->
		{:else if step === 'pick_child'}
			<div class="space-y-6">
				<div class="text-center">
					<span class="mx-auto grid size-16 place-items-center rounded-3xl bg-sun-tint text-3xl shadow-sm">
						🎮
					</span>
					<h2 class="mt-3 text-xl sm:text-2xl font-black text-ink">Siapa yang Mau Main?</h2>
					<p class="mt-1 text-xs sm:text-sm font-medium text-ink-soft">Pilih profilmu untuk mulai belajar dan bermain:</p>
				</div>

				{#if verifiedChildren.length === 0}
					<div class="rounded-3xl border-2 border-dashed border-line p-6 text-center">
						<p class="font-bold text-ink">Belum ada profil anak di akun ini.</p>
						<p class="mt-1 text-xs text-ink-soft">
							Minta Ayah atau Bunda masuk sebagai Orang Tua untuk membuat profil anak ya!
						</p>
						<button
							type="button"
							class="btn btn-primary btn-sm mt-4"
							onclick={() => {
								role = 'parent';
								step = 'phone';
								error = '';
							}}
						>
							Buka Mode Orang Tua
						</button>
					</div>
				{:else}
					<ul class="grid grid-cols-1 gap-3.5 sm:grid-cols-2">
						{#each verifiedChildren as c (c.id)}
							<li>
								<button
									type="button"
									class="press group flex w-full flex-col items-center gap-2.5 rounded-3xl border-2 border-line bg-white p-5 text-center transition-all hover:border-sun hover:bg-sun-tint/20 hover:shadow-md active:scale-95"
									onclick={() => chooseChild(c.id)}
								>
									<Avatar code={c.avatar} size={76} ring />
									<div class="min-w-0">
										<p class="truncate text-base sm:text-lg font-black text-ink">{c.nickname}</p>
										<p class="text-xs font-semibold text-ink-soft">{c.grade_name}</p>
									</div>
									<div class="flex items-center gap-1.5">
										<span class="pill bg-primary-tint text-primary-deep text-xs font-bold">Level {c.level}</span>
										<span class="pill bg-sun-tint text-sun-deep text-xs font-black">{c.xp} XP</span>
									</div>
									<span class="mt-1 text-xs font-black text-sun-deep group-hover:underline">
										Pilih & Main 🚀
									</span>
								</button>
							</li>
						{/each}
					</ul>
				{/if}
			</div>
		{/if}
	{/if}
</AuthShell>
