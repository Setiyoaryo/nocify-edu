<script lang="ts">
	import { get } from '$lib/api';
	import { page } from '$app/state';
	import type { ChildRecap } from '$lib/types';
	import Avatar from '$lib/components/Avatar.svelte';
	import CaretLeft from 'phosphor-svelte/lib/CaretLeft';
	import CaretRight from 'phosphor-svelte/lib/CaretRight';
	import Sparkle from 'phosphor-svelte/lib/Sparkle';
	import Star from 'phosphor-svelte/lib/Star';
	import Fire from 'phosphor-svelte/lib/Fire';
	import Lightning from 'phosphor-svelte/lib/Lightning';
	import Trophy from 'phosphor-svelte/lib/Trophy';
	import Crosshair from 'phosphor-svelte/lib/Crosshair';
	import Target from 'phosphor-svelte/lib/Target';
	import BookOpenText from 'phosphor-svelte/lib/BookOpenText';
	import InstagramLogo from 'phosphor-svelte/lib/InstagramLogo';
	import DownloadSimple from 'phosphor-svelte/lib/DownloadSimple';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';

	const id = page.params.childId as string;
	let recap = $state<ChildRecap | null>(null);
	let activeSlide = $state<0 | 1>(0);
	let loading = $state(true);
	let error = $state('');
	let toast = $state('');
	let exporting = $state(false);

	get(`/children/${id}/recap`)
		.then((r) => {
			recap = r;
			loading = false;
		})
		.catch((err) => {
			error = err.message || 'Gagal memuat rekap.';
			loading = false;
		});

	function showToast(msg: string) {
		toast = msg;
		setTimeout(() => {
			if (toast === msg) toast = '';
		}, 3500);
	}

	const tierTheme = $derived.by(() => {
		const t = recap?.archetype.tier;
		switch (t) {
			case 'legendary':
				return {
					border: 'border-amber-400 ring-4 ring-amber-300/60 shadow-[0_0_40px_rgba(251,191,36,0.35)]',
					badge: 'bg-gradient-to-r from-amber-500 to-yellow-400 text-ink',
					cardBg: 'bg-gradient-to-b from-[#fffbe6] via-[#fffdf0] to-[#fff8d6]',
					accent: 'text-amber-600',
					glow: 'from-amber-400/30 to-yellow-500/10'
				};
			case 'epic':
				return {
					border: 'border-purple-400 ring-4 ring-purple-300/50 shadow-[0_0_35px_rgba(168,85,247,0.3)]',
					badge: 'bg-gradient-to-r from-purple-600 to-indigo-500 text-white',
					cardBg: 'bg-gradient-to-b from-[#faf5ff] via-[#ffffff] to-[#f3e8ff]',
					accent: 'text-purple-600',
					glow: 'from-purple-500/20 to-indigo-500/10'
				};
			case 'rare':
				return {
					border: 'border-sky-400 ring-4 ring-sky-300/50 shadow-[0_0_30px_rgba(56,189,248,0.25)]',
					badge: 'bg-gradient-to-r from-sky-500 to-blue-500 text-white',
					cardBg: 'bg-gradient-to-b from-[#f0f9ff] via-[#ffffff] to-[#e0f2fe]',
					accent: 'text-sky-600',
					glow: 'from-sky-400/20 to-blue-500/10'
				};
			default:
				return {
					border: 'border-emerald-400 ring-4 ring-emerald-300/40 shadow-[0_0_25px_rgba(52,211,153,0.2)]',
					badge: 'bg-gradient-to-r from-emerald-500 to-teal-500 text-white',
					cardBg: 'bg-gradient-to-b from-[#f0fdf4] via-[#ffffff] to-[#dcfce7]',
					accent: 'text-emerald-600',
					glow: 'from-emerald-400/20 to-teal-500/10'
				};
		}
	});

	function wrapText(ctx: CanvasRenderingContext2D, text: string, x: number, y: number, maxWidth: number, lineHeight: number): number {
		const words = text.split(' ');
		let line = '';
		let curY = y;
		for (let n = 0; n < words.length; n++) {
			const testLine = line + words[n] + ' ';
			const metrics = ctx.measureText(testLine);
			if (metrics.width > maxWidth && n > 0) {
				ctx.fillText(line.trim(), x, curY);
				line = words[n] + ' ';
				curY += lineHeight;
			} else {
				line = testLine;
			}
		}
		ctx.fillText(line.trim(), x, curY);
		return curY;
	}

	function roundRect(ctx: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, r: number) {
		ctx.beginPath();
		ctx.moveTo(x + r, y);
		ctx.lineTo(x + w - r, y);
		ctx.arcTo(x + w, y, x + w, y + r, r);
		ctx.lineTo(x + w, y + h - r);
		ctx.arcTo(x + w, y + h, x + w - r, y + h, r);
		ctx.lineTo(x + r, y + h);
		ctx.arcTo(x, y + h, x, y + h - r, r);
		ctx.lineTo(x, y + r);
		ctx.arcTo(x, y, x + r, y, r);
		ctx.closePath();
	}

	async function generateCanvas(): Promise<Blob | null> {
		if (!recap) return null;
		const canvas = document.createElement('canvas');
		canvas.width = 1080;
		canvas.height = 1920;
		const ctx = canvas.getContext('2d');
		if (!ctx) return null;

		// 1. Dark outer space background
		const bg = ctx.createLinearGradient(0, 0, 1080, 1920);
		bg.addColorStop(0, '#0d0f28');
		bg.addColorStop(0.4, '#1b1d4a');
		bg.addColorStop(1, '#090a18');
		ctx.fillStyle = bg;
		ctx.fillRect(0, 0, 1080, 1920);

		// Background stars & lightning particles
		ctx.fillStyle = 'rgba(255, 255, 255, 0.2)';
		for (let i = 0; i < 40; i++) {
			const sx = (i * 137) % 1080;
			const sy = (i * 229) % 1920;
			const sr = ((i % 3) + 1) * 2;
			ctx.beginPath();
			ctx.arc(sx, sy, sr, 0, Math.PI * 2);
			ctx.fill();
		}

		// 2. Card Frame
		const cx = 70;
		const cy = 130;
		const cw = 940;
		const ch = 1660;
		const cr = 48;

		// Card Outer glow
		ctx.save();
		ctx.shadowColor = recap.archetype.tier === 'legendary' ? '#f59e0b' : '#a855f7';
		ctx.shadowBlur = 40;
		roundRect(ctx, cx, cy, cw, ch, cr);
		ctx.fillStyle = '#ffffff';
		ctx.fill();
		ctx.restore();

		// Card Background gradient
		const cardGrad = ctx.createLinearGradient(cx, cy, cx, cy + ch);
		if (recap.archetype.tier === 'legendary') {
			cardGrad.addColorStop(0, '#fffbeb');
			cardGrad.addColorStop(0.5, '#ffffff');
			cardGrad.addColorStop(1, '#fef3c7');
		} else if (recap.archetype.tier === 'epic') {
			cardGrad.addColorStop(0, '#faf5ff');
			cardGrad.addColorStop(0.5, '#ffffff');
			cardGrad.addColorStop(1, '#f3e8ff');
		} else {
			cardGrad.addColorStop(0, '#f0f9ff');
			cardGrad.addColorStop(0.5, '#ffffff');
			cardGrad.addColorStop(1, '#e0f2fe');
		}
		roundRect(ctx, cx, cy, cw, ch, cr);
		ctx.fillStyle = cardGrad;
		ctx.fill();

		// Card Border
		ctx.lineWidth = 14;
		ctx.strokeStyle = recap.archetype.tier === 'legendary' ? '#fbbf24' : recap.archetype.tier === 'epic' ? '#c084fc' : '#38bdf8';
		ctx.stroke();

		// 3. Brand & Tier Header
		ctx.textAlign = 'center';
		ctx.font = '900 42px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#1b1d3a';
		ctx.fillText('⚡ EDUPPLAY ⚡', 540, 225);

		// Tier Pill
		const pillW = 320;
		const pillH = 50;
		const pillX = 540 - pillW / 2;
		const pillY = 250;
		roundRect(ctx, pillX, pillY, pillW, pillH, 25);
		ctx.fillStyle = recap.archetype.tier === 'legendary' ? '#f59e0b' : recap.archetype.tier === 'epic' ? '#9333ea' : '#0284c7';
		ctx.fill();

		ctx.font = '800 24px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#ffffff';
		ctx.fillText(`⭐ TIER ${recap.archetype.tier_name.toUpperCase()} ⭐`, 540, pillY + 34);

		// 4. Archetype Title Banner
		ctx.font = '900 50px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#1b1d3a';
		ctx.fillText(recap.archetype.title, 540, 370);

		ctx.font = '700 26px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#5c6079';
		ctx.fillText(`${recap.archetype.tag.toUpperCase()} • ${recap.archetype.subtitle}`, 540, 415);

		// 5. Character Ring Stage
		const avY = 640;
		// Outer decorative halo ring
		ctx.beginPath();
		ctx.arc(540, avY, 150, 0, Math.PI * 2);
		ctx.fillStyle = recap.archetype.tier === 'legendary' ? 'rgba(251,191,36,0.18)' : 'rgba(192,132,252,0.18)';
		ctx.fill();
		ctx.lineWidth = 6;
		ctx.strokeStyle = recap.archetype.tier === 'legendary' ? '#f59e0b' : '#9333ea';
		ctx.stroke();

		// Inner disc
		ctx.beginPath();
		ctx.arc(540, avY, 125, 0, Math.PI * 2);
		ctx.fillStyle = '#ffffff';
		ctx.fill();
		ctx.shadowColor = 'rgba(0,0,0,0.1)';
		ctx.shadowBlur = 15;

		// Draw character badge in center
		ctx.font = '900 72px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#5b5ce2';
		ctx.fillText(recap.child.nickname.slice(0, 2).toUpperCase(), 540, avY + 22);

		// Level Pill under character
		const lvlW = 260;
		const lvlH = 46;
		roundRect(ctx, 540 - lvlW / 2, avY + 110, lvlW, lvlH, 23);
		ctx.fillStyle = '#1b1d3a';
		ctx.fill();
		ctx.font = '800 24px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#ffc857';
		ctx.fillText(`Level ${recap.child.level} • ${recap.child.nickname}`, 540, avY + 142);

		// 6. Stats Grid (4 Boxes)
		const statY = 880;
		const statW = 205;
		const statH = 135;
		const statGap = 16;
		const startX = cx + 30;

		const statsList = [
			{ icon: '🔥', val: `${recap.stats.streak} Hari`, label: 'Streak' },
			{ icon: '🏆', val: `${recap.stats.total_challenges}`, label: 'Tantangan' },
			{ icon: '⭐', val: `${recap.stats.stars}`, label: 'Bintang' },
			{ icon: '🎯', val: `${Math.round(recap.stats.accuracy)}%`, label: 'Akurasi' }
		];

		statsList.forEach((s, idx) => {
			const bx = startX + idx * (statW + statGap);
			roundRect(ctx, bx, statY, statW, statH, 22);
			ctx.fillStyle = '#ffffff';
			ctx.fill();
			ctx.lineWidth = 3;
			ctx.strokeStyle = '#e4e7f0';
			ctx.stroke();

			ctx.font = '30px sans-serif';
			ctx.fillText(s.icon, bx + statW / 2, statY + 42);
			ctx.font = '900 28px "Plus Jakarta Sans", system-ui, sans-serif';
			ctx.fillStyle = '#1b1d3a';
			ctx.fillText(s.val, bx + statW / 2, statY + 84);
			ctx.font = '700 19px "Plus Jakarta Sans", system-ui, sans-serif';
			ctx.fillStyle = '#5c6079';
			ctx.fillText(s.label, bx + statW / 2, statY + 115);
		});

		// 7. Praise Speech Bubble Card
		const bubbleY = 1070;
		const bubbleW = 860;
		const bubbleH = 430;
		const bubbleX = 540 - bubbleW / 2;

		roundRect(ctx, bubbleX, bubbleY, bubbleW, bubbleH, 32);
		ctx.fillStyle = '#ffffff';
		ctx.fill();
		ctx.lineWidth = 4;
		ctx.strokeStyle = recap.archetype.tier === 'legendary' ? '#fcd34d' : '#e4e7f0';
		ctx.stroke();

		// Quotation mark
		ctx.font = '900 50px serif';
		ctx.fillStyle = '#ffc857';
		ctx.fillText('“', bubbleX + 50, bubbleY + 65);

		// Praise text
		ctx.font = '600 32px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#1b1d3a';
		const nextY = wrapText(ctx, recap.archetype.praise, 540, bubbleY + 110, 760, 48);

		// Next tier hint inside praise card
		roundRect(ctx, bubbleX + 40, bubbleY + bubbleH - 100, bubbleW - 80, 68, 20);
		ctx.fillStyle = '#f5f6fb';
		ctx.fill();

		ctx.font = '700 22px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#5b5ce2';
		ctx.fillText(`🚀 Target Berikutnya: ${recap.archetype.next_tier_hint}`, 540, bubbleY + bubbleH - 58);

		// 8. Watermark Footer
		ctx.font = '700 26px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#8a8fa8';
		ctx.fillText('EDUPPLAY • Petualangan Belajar TK–SD • eduplay.id', 540, cy + ch - 50);

		return new Promise((resolve) => canvas.toBlob(resolve, 'image/png'));
	}

	async function downloadImage() {
		if (exporting || !recap) return;
		exporting = true;
		try {
			const blob = await generateCanvas();
			if (!blob) throw new Error('Gagal merender kartu.');
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `kartu-legendaris-${recap.child.nickname.toLowerCase()}.png`;
			document.body.appendChild(a);
			a.click();
			document.body.removeChild(a);
			URL.revokeObjectURL(url);
			showToast('Gambar kartu berhasil diunduh! Siap dipamerkan 🎉');
		} catch (e: any) {
			showToast(e.message || 'Gagal mengunduh.');
		} finally {
			exporting = false;
		}
	}

	async function shareToInstagram() {
		if (exporting || !recap) return;
		exporting = true;
		try {
			const blob = await generateCanvas();
			if (!blob) throw new Error('Gagal merender kartu.');
			const file = new File([blob], `kartu-legendaris-${recap.child.nickname.toLowerCase()}.png`, { type: 'image/png' });

			if (navigator.canShare && navigator.canShare({ files: [file] })) {
				await navigator.share({
					files: [file],
					title: `Kartu Legendaris ${recap.child.nickname} di EduPlay`,
					text: `Lihat kartu ${recap.archetype.title} milik ${recap.child.nickname}! Level ${recap.child.level} dengan ${recap.stats.stars} bintang di EduPlay.`
				});
				showToast('Berhasil membuka menu bagikan! 🚀');
			} else {
				// Fallback: download the file & copy story text
				await downloadImage();
				if (navigator.clipboard) {
					await navigator.clipboard.writeText(`Lihat kartu karakter ${recap.child.nickname} di EduPlay! Gelar: ${recap.archetype.title} 🔥`);
				}
				showToast('Gambar tersimpan! Buka Instagram Story untuk membagikannya ✨');
			}
		} catch (e: any) {
			if (e.name !== 'AbortError') {
				showToast('Buka fitur bagikan atau unduh gambar secara manual.');
			}
		} finally {
			exporting = false;
		}
	}
</script>

<svelte:head>
	<title>{recap ? `Kartu Legendaris ${recap.child.nickname}` : 'Kartu Legendaris'} | EduPlay</title>
</svelte:head>

<div class="relative mx-auto max-w-lg px-4 py-4 sm:py-6">
	<!-- Top Bar -->
	<header class="flex items-center justify-between pb-3">
		<a href="/app/play/{id}" class="inline-flex items-center gap-1 text-sm font-bold text-ink-soft hover:text-ink">
			<CaretLeft size={20} weight="bold" aria-hidden="true" />
			Kembali
		</a>
		<span class="inline-flex items-center gap-1.5 rounded-full bg-paper px-3 py-1 text-xs font-extrabold shadow-sm">
			<Sparkle size={16} weight="fill" class="text-sun-deep" aria-hidden="true" />
			Koleksi Legenda
		</span>
	</header>

	{#if toast}
		<div class="anim-nudge fixed top-5 left-1/2 z-50 -translate-x-1/2 rounded-full bg-night px-6 py-3 text-sm font-extrabold text-white shadow-lift" role="status">
			{toast}
		</div>
	{/if}

	{#if loading}
		<div class="mt-8 flex flex-col items-center justify-center gap-4 py-24 text-center">
			<div class="size-12 animate-spin rounded-full border-4 border-primary border-t-transparent"></div>
			<p class="font-bold text-ink-soft">Mempersiapkan kartu legendarismu…</p>
		</div>
	{:else if error || !recap}
		<div class="mt-8 rounded-2xl bg-paper p-6 text-center shadow-card">
			<p class="font-bold text-error">{error || 'Kartu belum tersedia.'}</p>
			<a href="/app/play/{id}" class="btn btn-primary mt-4">Kembali bermain</a>
		</div>
	{:else}
		<!-- Main Card Preview Stage -->
		<div class="mt-2 flex flex-col items-center">
			{#if activeSlide === 0}
				<!-- SLIDE 1: KARTU KARAKTER LEGENDARIS (TCG FRAME) -->
				<article
					class="relative w-full overflow-hidden rounded-[2.2rem] border-4 p-6 sm:p-7 {tierTheme.border} {tierTheme.cardBg} transition-all duration-300"
					style="min-height: 520px;"
				>
					<!-- Glowing aura background -->
					<div class="pointer-events-none absolute -top-12 -right-12 size-48 rounded-full bg-gradient-to-br {tierTheme.glow} blur-2xl"></div>
					<div class="pointer-events-none absolute -bottom-12 -left-12 size-48 rounded-full bg-gradient-to-tr {tierTheme.glow} blur-2xl"></div>

					<!-- Header Brand & Tier Badge -->
					<div class="relative flex items-center justify-between">
						<span class="font-display text-sm font-black tracking-wider text-ink/80">⚡ EDUPPLAY</span>
						<span class="inline-flex items-center gap-1 rounded-full px-3 py-1 text-xs font-black tracking-wide uppercase shadow-sm {tierTheme.badge}">
							<Star size={14} weight="fill" aria-hidden="true" />
							{recap.archetype.tier_name}
						</span>
					</div>

					<!-- Title & Subtitle -->
					<div class="relative mt-4 text-center">
						<h1 class="font-display text-2xl font-black tracking-tight text-ink sm:text-3xl">
							⚡ {recap.archetype.title} ⚡
						</h1>
						<p class="mt-1 text-xs font-bold uppercase tracking-wider text-ink-soft sm:text-sm">
							{recap.archetype.tag} • Level {recap.child.level}
						</p>
					</div>

					<!-- Character Hero Disc -->
					<div class="relative mt-5 flex justify-center">
						<div class="relative flex size-32 items-center justify-center rounded-full bg-white shadow-lift ring-8 ring-amber-300/40">
							<Avatar code={recap.child.avatar} size={90} />
							<!-- Floating badge particles -->
							<span class="absolute -top-1 -right-1 grid size-8 place-items-center rounded-full bg-amber-400 text-ink shadow-sm">
								<Sparkle size={18} weight="fill" />
							</span>
							{#if recap.stats.streak >= 3}
								<span class="absolute -bottom-1 -left-1 grid size-8 place-items-center rounded-full bg-tangerine text-white shadow-sm">
									<Fire size={18} weight="fill" />
								</span>
							{/if}
						</div>
					</div>

					<!-- Nickname Badge -->
					<div class="relative mt-3 text-center">
						<span class="inline-block rounded-full bg-night px-4 py-1 text-xs font-black text-sun">
							{recap.child.nickname} • Kelas {recap.child.grade_name}
						</span>
					</div>

					<!-- 4 Stat Badges Row -->
					<div class="relative mt-5 grid grid-cols-4 gap-2 text-center">
						<div class="rounded-xl border border-line bg-white/90 p-2 shadow-xs">
							<span class="block text-base sm:text-lg">🔥</span>
							<b class="block text-sm font-black text-ink">{recap.stats.streak}</b>
							<span class="text-[0.65rem] font-bold text-ink-soft">Streak</span>
						</div>
						<div class="rounded-xl border border-line bg-white/90 p-2 shadow-xs">
							<span class="block text-base sm:text-lg">🏆</span>
							<b class="block text-sm font-black text-ink">{recap.stats.total_challenges}</b>
							<span class="text-[0.65rem] font-bold text-ink-soft">Tantangan</span>
						</div>
						<div class="rounded-xl border border-line bg-white/90 p-2 shadow-xs">
							<span class="block text-base sm:text-lg">⭐</span>
							<b class="block text-sm font-black text-ink">{recap.stats.stars}</b>
							<span class="text-[0.65rem] font-bold text-ink-soft">Bintang</span>
						</div>
						<div class="rounded-xl border border-line bg-white/90 p-2 shadow-xs">
							<span class="block text-base sm:text-lg">🎯</span>
							<b class="block text-sm font-black text-ink">{Math.round(recap.stats.accuracy)}%</b>
							<span class="text-[0.65rem] font-bold text-ink-soft">Akurasi</span>
						</div>
					</div>

					<!-- Praise Card Box (FitHub style speech quote) -->
					<div class="relative mt-4 rounded-2xl border border-amber-200/80 bg-white/95 p-4 text-center shadow-xs">
						<p class="text-xs font-bold leading-relaxed text-ink sm:text-sm">
							"{recap.archetype.praise}"
						</p>
					</div>

					<!-- Next Milestone Hint -->
					<div class="relative mt-3 text-center">
						<p class="text-[0.7rem] font-extrabold text-primary-deep sm:text-xs">
							🚀 {recap.archetype.next_tier_hint}
						</p>
					</div>
				</article>
			{:else}
				<!-- SLIDE 2: RAPOR PETUALANGAN LENGKAP -->
				<article
					class="relative w-full overflow-hidden rounded-[2.2rem] border-4 border-line bg-white p-6 shadow-lift sm:p-7 transition-all duration-300"
					style="min-height: 520px;"
				>
					<div class="flex items-center justify-between border-b border-line pb-3">
						<span class="font-display text-sm font-black text-ink">RAPOR PETUALANGAN</span>
						<span class="pill bg-sun-tint text-xs font-black text-ink">
							Level {recap.child.level} ({recap.child.xp} XP)
						</span>
					</div>

					<div class="mt-4 flex items-center gap-3">
						<Avatar code={recap.child.avatar} size={54} ring />
						<div>
							<h2 class="text-xl font-black text-ink">{recap.child.nickname}</h2>
							<p class="text-xs font-semibold text-ink-soft">{recap.child.grade_name} • {recap.archetype.title}</p>
						</div>
					</div>

					<!-- 6 Tiles Activity Breakdown -->
					<div class="mt-5 grid grid-cols-2 gap-3 text-left">
						<div class="rounded-2xl bg-paper-deep p-3.5">
							<span class="flex items-center gap-1.5 text-xs font-bold text-ink-soft">
								<Trophy size={16} class="text-sun-deep" weight="fill" />
								Tantangan
							</span>
							<p class="mt-1 text-lg font-black text-ink">{recap.stats.total_challenges} <span class="text-xs font-bold text-ink-soft">selesai</span></p>
						</div>

						<div class="rounded-2xl bg-paper-deep p-3.5">
							<span class="flex items-center gap-1.5 text-xs font-bold text-ink-soft">
								<CheckCircle size={16} class="text-leaf-deep" weight="fill" />
								Soal Dijawab
							</span>
							<p class="mt-1 text-lg font-black text-ink">{recap.stats.total_questions} <span class="text-xs font-bold text-ink-soft">soal</span></p>
						</div>

						<div class="rounded-2xl bg-paper-deep p-3.5">
							<span class="flex items-center gap-1.5 text-xs font-bold text-ink-soft">
								<Crosshair size={16} class="text-primary" weight="fill" />
								Akurasi Belajar
							</span>
							<p class="mt-1 text-lg font-black text-ink">{Math.round(recap.stats.accuracy)}% <span class="text-xs font-bold text-ink-soft">tepat</span></p>
						</div>

						<div class="rounded-2xl bg-paper-deep p-3.5">
							<span class="flex items-center gap-1.5 text-xs font-bold text-ink-soft">
								<Fire size={16} class="text-tangerine" weight="fill" />
								Konsistensi
							</span>
							<p class="mt-1 text-lg font-black text-ink">{recap.stats.streak} <span class="text-xs font-bold text-ink-soft">hari beruntun</span></p>
						</div>

						<div class="rounded-2xl bg-paper-deep p-3.5">
							<span class="flex items-center gap-1.5 text-xs font-bold text-ink-soft">
								<BookOpenText size={16} class="text-rose" weight="fill" />
								Paling Disukai
							</span>
							<p class="mt-1 truncate text-base font-black text-ink">{recap.stats.favorite_subject || 'Semua Pelajaran'}</p>
						</div>

						<div class="rounded-2xl bg-paper-deep p-3.5">
							<span class="flex items-center gap-1.5 text-xs font-bold text-ink-soft">
								<Star size={16} class="text-sun-deep" weight="fill" />
								Lencana Terbaik
							</span>
							<p class="mt-1 truncate text-base font-black text-ink">{recap.stats.top_badge?.name || 'Penjelajah Cilik'}</p>
						</div>
					</div>

					<div class="mt-6 rounded-2xl bg-primary-tint p-4 text-center">
						<p class="text-xs font-extrabold text-primary-deep">
							🌟 Terus bermain tantangan baru untuk mengoleksi lencana dan menaikkan tier ke Legendaris!
						</p>
					</div>
				</article>
			{/if}

			<!-- Carousel Dots Indicator (matching FitHub reference) -->
			<div class="mt-5 flex items-center justify-center gap-2" role="tablist" aria-label="Pilih Slide">
				<button
					role="tab"
					aria-selected={activeSlide === 0}
					aria-label="Kartu Karakter"
					onclick={() => (activeSlide = 0)}
					class="transition-all duration-300 {activeSlide === 0 ? 'h-2.5 w-8 rounded-full bg-sun' : 'size-2.5 rounded-full bg-line hover:bg-ink-soft'}"
				></button>
				<button
					role="tab"
					aria-selected={activeSlide === 1}
					aria-label="Rapor Petualangan"
					onclick={() => (activeSlide = 1)}
					class="transition-all duration-300 {activeSlide === 1 ? 'h-2.5 w-8 rounded-full bg-sun' : 'size-2.5 rounded-full bg-line hover:bg-ink-soft'}"
				></button>
			</div>

			<!-- Action Buttons (Bagikan & Unduh) -->
			<div class="mt-6 flex w-full flex-col gap-3">
				<button
					class="btn btn-sun w-full shadow-card text-base py-3.5"
					onclick={shareToInstagram}
					disabled={exporting}
				>
					<InstagramLogo size={22} weight="fill" aria-hidden="true" />
					{exporting ? 'Memproses Kartu…' : 'Bagikan di Instagram / Story'}
				</button>

				<button
					class="inline-flex items-center justify-center gap-2 py-3 text-sm font-extrabold text-ink-soft hover:text-ink transition"
					onclick={downloadImage}
					disabled={exporting}
				>
					<DownloadSimple size={20} weight="bold" aria-hidden="true" />
					Unduh Gambar (PNG HD)
				</button>
			</div>
		</div>
	{/if}
</div>
