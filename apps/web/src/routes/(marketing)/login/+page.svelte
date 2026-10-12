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
	import Star from 'phosphor-svelte/lib/Star';
	import Lightbulb from 'phosphor-svelte/lib/Lightbulb';
	import Backpack from 'phosphor-svelte/lib/Backpack';
	import WarningCircle from 'phosphor-svelte/lib/WarningCircle';
	import Key from 'phosphor-svelte/lib/Key';
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

	$effect(() => {
		const paramRole = page.url.searchParams.get('role');
		if (paramRole === 'parent' || paramRole === 'child') {
			role = paramRole;
		} else {
			role = null;
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
		if (page.url.searchParams.get('role') !== r) {
			const u = new URL(page.url);
			u.searchParams.set('role', r);
			goto(u.pathname + u.search, { replaceState: false, noScroll: true, keepFocus: true });
		}
	}

	function resetRole() {
		role = null;
		step = 'phone';
		error = '';
		otpDigits = ['', '', '', '', '', ''];
		if (page.url.searchParams.has('role')) {
			const u = new URL(page.url);
			u.searchParams.delete('role');
			goto(u.pathname + (u.search ? u.search : ''), { replaceState: false, noScroll: true, keepFocus: true });
		}
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
			const res = await post<{ status: string; phone: string; message: string }>('/auth/otp/send', {
				phone: cleanPhone,
				role: role ?? 'parent'
			});
			formattedPhone = res.phone;
			step = 'otp';
			startTimer(60);
			queueMicrotask(() => {
				if (!document.activeElement?.id?.startsWith('otp-')) {
					document.getElementById('otp-0')?.focus();
				}
			});
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Gagal mengirim kode OTP. Coba beberapa saat lagi.';
		} finally {
			busy = false;
		}
	}

	function handleDigitInput(index: number, e: Event) {
		const target = e.target as HTMLInputElement;
		const val = target.value.replace(/\D/g, '');
		error = '';

		if (val.length > 1) {
			// Handle mobile SMS autofill where full code (e.g. 6 digits) is inserted into active field
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
	title={role === 'child'
		? 'Ruang Bermain Anak'
		: role === 'parent'
			? 'Masuk Area Orang Tua'
			: 'Masuk sebagai siapa?'}
	lead={role === 'child'
		? 'Ayo sambung petualanganmu, selesaikan tantangan seru, dan kumpulkan bintang hari ini!'
		: role === 'parent'
			? 'Masuk untuk memantau progres belajar anak, mengatur misi, dan mengelola akun.'
			: 'Pilih peran Anda untuk pengalaman belajar yang tepat dan menyenangkan.'}
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
		<div class="mt-2 sm:mt-3">
			<div class="grid gap-3.5 sm:gap-4 sm:grid-cols-2">
				<!-- Orang Tua Card -->
				<a
					href="/login?role=parent"
					role="button"
					class="press group relative flex flex-col items-start gap-3 sm:gap-4 rounded-3xl border-2 border-line bg-white p-4.5 sm:p-5 text-left transition-all hover:border-primary hover:bg-primary-tint/20 hover:shadow-md active:scale-[0.98]"
					onclick={(e) => {
						e.preventDefault();
						selectRole('parent');
					}}
				>
					<div class="flex w-full items-center justify-between">
						<span
							class="grid size-12 sm:size-13 place-items-center rounded-2xl bg-primary-tint text-primary-deep shadow-sm transition-transform group-hover:scale-105"
						>
							<UsersThree size={28} weight="fill" aria-hidden="true" />
						</span>
						<span class="pill bg-primary/10 px-2.5 py-1 text-xs font-bold text-primary-deep">Area Orang Tua</span>
					</div>
					<div>
						<h2 class="text-lg font-extrabold text-ink sm:text-xl">Orang Tua</h2>
						<p class="mt-1 text-xs sm:text-sm leading-relaxed text-ink-soft">
							Pantau laporan skill, buat misi mingguan, dan kelola akun anak.
						</p>
					</div>
					<div class="mt-auto flex w-full items-center justify-between pt-2 text-xs sm:text-sm font-bold text-primary-deep">
						<span>Masuk Orang Tua</span>
						<ArrowRight size={16} weight="bold" class="transition-transform group-hover:translate-x-1" aria-hidden="true" />
					</div>
				</a>

				<!-- Anak Card (Kid Friendly) -->
				<a
					href="/login?role=child"
					role="button"
					class="press group relative flex flex-col items-start gap-3 sm:gap-4 rounded-3xl border-2 border-sun/60 bg-gradient-to-b from-white to-sun-tint/30 p-4.5 sm:p-5 text-left transition-all hover:border-sun hover:bg-sun-tint/40 hover:shadow-md active:scale-[0.98]"
					onclick={(e) => {
						e.preventDefault();
						selectRole('child');
					}}
				>
					<div class="flex w-full items-center justify-between">
						<span
							class="grid size-12 sm:size-13 place-items-center rounded-2xl bg-sun-tint text-sun-deep shadow-sm transition-transform group-hover:scale-105"
						>
							<GameController size={28} weight="fill" aria-hidden="true" />
						</span>
						<span class="pill inline-flex items-center gap-1.5 bg-sun/20 px-2.5 py-1 text-xs font-black tracking-wide text-sun-deep">
							<Star size={13} weight="fill" aria-hidden="true" />
							<span>Ruang Bermain</span>
						</span>
					</div>
					<div>
						<div class="flex items-center gap-2">
							<h2 class="text-lg font-black text-ink sm:text-xl">Anak</h2>
							<span class="rounded-lg bg-sun/20 px-1.5 py-0.5 text-[10px] font-extrabold text-sun-deep">Akun dari Orang Tua</span>
						</div>
						<p class="mt-1 text-xs sm:text-sm leading-relaxed text-ink-soft">
							Mulai petualangan kuis seru, raih bintang, dan koleksi avatar lucu!
						</p>
					</div>
					<div class="mt-auto flex w-full items-center justify-between pt-2 text-xs sm:text-sm font-black text-sun-deep">
						<span>Ayo Main Sekarang!</span>
						<Rocket size={16} weight="fill" class="transition-transform group-hover:translate-x-1" aria-hidden="true" />
					</div>
				</a>
			</div>

			<!-- Notice about child accounts created by parents -->
			<div class="mt-4 sm:mt-5 flex items-start gap-2.5 sm:gap-3 rounded-2xl border border-line bg-paper-deep p-3.5 text-xs text-ink-soft">
				<span class="grid size-6 place-items-center rounded-lg bg-sun/30 text-sun-deep shrink-0 mt-0.5">
					<Lightbulb size={16} weight="fill" aria-hidden="true" />
				</span>
				<p class="leading-relaxed">
					<strong class="text-ink font-bold">Catatan untuk Anak:</strong> Anak dapat login setelah akun dan profilnya dibuatkan oleh Orang Tua. Pastikan Ayah atau Bunda sudah mendaftar terlebih dahulu ya!
				</p>
			</div>

			<p class="mt-6 sm:mt-8 text-center text-xs sm:text-sm text-ink-soft">
				Belum punya akun? <a class="font-bold text-primary-deep hover:underline" href="/register">Daftar gratis</a>
			</p>
		</div>

	<!-- STAGE 2: FORM FOR SELECTED ROLE -->
	{:else}
		<!-- Header with role switch and back button -->
		<div class="mt-1 mb-5 sm:mb-6 flex flex-wrap items-center justify-between gap-2 border-b border-line pb-3.5 sm:pb-4">
			<a
				href="/login"
				role="button"
				class="inline-flex items-center gap-1.5 text-xs sm:text-sm font-bold text-ink-soft hover:text-ink transition-colors shrink-0"
				onclick={(e) => {
					e.preventDefault();
					resetRole();
				}}
			>
				<ArrowLeft size={16} weight="bold" aria-hidden="true" />
				<span>Ganti peran</span>
			</a>

			<div class="flex items-center rounded-2xl bg-paper-deep p-1 shadow-inner shrink-0" role="tablist">
				<a
					href="/login?role=parent"
					role="tab"
					aria-selected={role === 'parent'}
					class="flex items-center gap-1.5 rounded-xl px-2.5 sm:px-3 py-1.5 text-xs sm:text-sm font-bold transition-all {role === 'parent'
						? 'bg-white text-primary-deep shadow-sm'
						: 'text-ink-soft hover:text-ink'}"
					onclick={(e) => {
						e.preventDefault();
						selectRole('parent');
					}}
				>
					<UsersThree size={16} weight="fill" aria-hidden="true" />
					<span>Orang Tua</span>
				</a>
				<a
					href="/login?role=child"
					role="tab"
					aria-selected={role === 'child'}
					class="flex items-center gap-1.5 rounded-xl px-2.5 sm:px-3 py-1.5 text-xs sm:text-sm font-bold transition-all {role === 'child'
						? 'bg-sun text-ink shadow-sm'
						: 'text-ink-soft hover:text-ink'}"
					onclick={(e) => {
						e.preventDefault();
						selectRole('child');
					}}
				>
					<GameController size={16} weight="fill" aria-hidden="true" />
					<span>Anak</span>
				</a>
			</div>
		</div>

		<!-- STEP 1: PHONE INPUT -->
		{#if step === 'phone'}
			<div class="space-y-4 sm:space-y-5">
				<div>
					<h2 class="text-xl sm:text-2xl font-extrabold text-ink flex items-center gap-2 flex-wrap">
						{#if role === 'child'}
							<span>Hai Petualang Cilik!</span>
							<span class="anim-float inline-flex items-center gap-1">
								<Backpack size={24} weight="fill" class="text-primary-deep" aria-hidden="true" />
								<Sparkle size={20} weight="fill" class="text-sun-deep" aria-hidden="true" />
							</span>
						{:else}
							<span>Masuk sebagai Orang Tua</span>
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
					<div class="flex items-start gap-2.5 sm:gap-3 rounded-2xl border border-sun/50 bg-sun-tint/50 p-3.5 text-xs sm:text-sm text-ink shadow-sm">
						<span class="grid size-7 place-items-center rounded-xl bg-sun/40 text-sun-deep shrink-0 mt-0.5">
							<Backpack size={18} weight="fill" aria-hidden="true" />
						</span>
						<div class="space-y-1">
							<p class="leading-relaxed font-black text-ink">
								Akun anak dibuatkan oleh Orang Tua
							</p>
							<p class="text-xs text-ink-soft leading-relaxed">
								Ketik nomor WhatsApp Ayah atau Bunda yang sudah didaftarkan. Kode rahasia 6 angka akan dikirimkan ke WhatsApp mereka untuk langsung bermain!
							</p>
						</div>
					</div>
				{/if}

				<form onsubmit={submitSendOTP} novalidate class="space-y-4 sm:space-y-5">
					<div>
						<label class="label text-xs sm:text-sm" for="phone">
							{role === 'child' ? 'Nomor WhatsApp Ayah / Bunda' : 'Nomor WhatsApp'}
						</label>
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
								bind:value={phone}
							/>
						</div>
						<p class="mt-1.5 text-xs text-ink-soft">
							{role === 'child' ? 'Bisa ketik 08... atau 8...' : 'Bisa diawali 08 atau langsung 8 (contoh: 081234567890)'}
						</p>
					</div>

					{#if error}
						<div class="rounded-2xl border border-red-200 bg-red-50 p-3.5 text-xs sm:text-sm text-red-800 shadow-sm" role="alert">
							<div class="flex items-start gap-2.5">
								<span class="grid size-6 place-items-center rounded-lg bg-red-100 text-red-700 shrink-0 mt-0.5">
									<WarningCircle size={18} weight="fill" aria-hidden="true" />
								</span>
								<div class="flex-1 space-y-1.5">
									<p class="font-bold">{error}</p>
									{#if role === 'child'}
										<p class="text-xs text-red-700 leading-relaxed">
											Anak hanya bisa masuk jika akun dan profil anak sudah dibuat oleh orang tua terlebih dahulu.
										</p>
										<div class="mt-2 flex flex-wrap gap-2 pt-1">
											<button
												type="button"
												class="press inline-flex items-center gap-1 rounded-xl bg-white px-2.5 py-1.5 text-xs font-bold text-ink border border-red-200 shadow-xs hover:bg-paper-deep"
												onclick={() => selectRole('parent')}
											>
												<UsersThree size={14} weight="fill" />
												<span>Masuk Mode Orang Tua</span>
											</button>
											<a
												href="/register"
												class="press inline-flex items-center gap-1 rounded-xl bg-primary px-2.5 py-1.5 text-xs font-bold text-white shadow-xs hover:bg-primary-deep"
											>
												<span>Daftar Akun Baru</span>
											</a>
										</div>
									{/if}
								</div>
							</div>
						</div>
					{/if}

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
							<span>Kirim Kode ke Ayah / Bunda</span>
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
					<h2 class="text-xl sm:text-2xl font-extrabold text-ink inline-flex items-center gap-2">
						<span>{role === 'child' ? 'Ketik Kode Rahasia' : 'Verifikasi Kode OTP'}</span>
						{#if role === 'child'}
							<Key size={22} weight="fill" class="text-sun-deep" aria-hidden="true" />
						{/if}
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

				<form onsubmit={submitVerifyOTP} novalidate class="space-y-5">
					<div>
						<label class="label block text-center text-xs sm:text-sm" for="otp-0">
							Masukkan 6 Digit Kode
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
									onpaste={handlePaste}
									autocomplete={i === 0 ? 'one-time-code' : 'off'}
								/>
							{/each}
						</div>
					</div>

					{#if error}
						<div class="rounded-2xl border border-red-200 bg-red-50 p-3 text-xs sm:text-sm text-red-800 text-center font-bold" role="alert">
							{error}
						</div>
					{/if}

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
							<span>Ayo Masuk & Main!</span>
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
					<span class="mx-auto grid size-16 place-items-center rounded-3xl bg-sun-tint text-sun-deep shadow-sm">
						<GameController size={36} weight="fill" aria-hidden="true" />
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
							onclick={() => selectRole('parent')}
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
									<span class="mt-1 inline-flex items-center gap-1 text-xs font-black text-sun-deep group-hover:underline">
										<span>Pilih & Main</span>
										<Rocket size={14} weight="fill" class="transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
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
