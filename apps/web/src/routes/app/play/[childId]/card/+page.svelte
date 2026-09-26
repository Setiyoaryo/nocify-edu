<script lang="ts">
	import { get } from '$lib/api';
	import { page } from '$app/state';
	import type { ChildRecap } from '$lib/types';
	import Avatar from '$lib/components/Avatar.svelte';
	import CaretLeft from 'phosphor-svelte/lib/CaretLeft';
	import Sparkle from 'phosphor-svelte/lib/Sparkle';
	import Star from 'phosphor-svelte/lib/Star';
	import Fire from 'phosphor-svelte/lib/Fire';
	import Trophy from 'phosphor-svelte/lib/Trophy';
	import Crosshair from 'phosphor-svelte/lib/Crosshair';
	import Crown from 'phosphor-svelte/lib/Crown';
	import ShieldCheck from 'phosphor-svelte/lib/ShieldCheck';
	import Medal from 'phosphor-svelte/lib/Medal';
	import Target from 'phosphor-svelte/lib/Target';
	import BookOpenText from 'phosphor-svelte/lib/BookOpenText';
	import InstagramLogo from 'phosphor-svelte/lib/InstagramLogo';
	import DownloadSimple from 'phosphor-svelte/lib/DownloadSimple';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	import Quotes from 'phosphor-svelte/lib/Quotes';
	import winnerArt from '$lib/illustrations/winner.svg';

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

	const tierConfig = $derived.by(() => {
		const t = recap?.archetype.tier;
		switch (t) {
			case 'legendary':
				return {
					border: 'border-amber-400 ring-4 ring-amber-300/60 shadow-[0_0_40px_rgba(251,191,36,0.35)]',
					badge: 'bg-gradient-to-r from-amber-500 to-yellow-400 text-ink',
					cardBg: 'bg-gradient-to-b from-[#fffbe6] via-[#fffdf0] to-[#fff8d6]',
					accent: 'text-amber-600',
					halo: 'ring-amber-400/50 bg-amber-50',
					icon: Crown
				};
			case 'epic':
				return {
					border: 'border-purple-400 ring-4 ring-purple-300/50 shadow-[0_0_35px_rgba(168,85,247,0.3)]',
					badge: 'bg-gradient-to-r from-purple-600 to-indigo-500 text-white',
					cardBg: 'bg-gradient-to-b from-[#faf5ff] via-[#ffffff] to-[#f3e8ff]',
					accent: 'text-purple-600',
					halo: 'ring-purple-400/50 bg-purple-50',
					icon: Medal
				};
			case 'rare':
				return {
					border: 'border-sky-400 ring-4 ring-sky-300/50 shadow-[0_0_30px_rgba(56,189,248,0.25)]',
					badge: 'bg-gradient-to-r from-sky-500 to-blue-500 text-white',
					cardBg: 'bg-gradient-to-b from-[#f0f9ff] via-[#ffffff] to-[#e0f2fe]',
					accent: 'text-sky-600',
					halo: 'ring-sky-400/50 bg-sky-50',
					icon: Star
				};
			default:
				return {
					border: 'border-emerald-400 ring-4 ring-emerald-300/40 shadow-[0_0_25px_rgba(52,211,153,0.2)]',
					badge: 'bg-gradient-to-r from-emerald-500 to-teal-500 text-white',
					cardBg: 'bg-gradient-to-b from-[#f0fdf4] via-[#ffffff] to-[#dcfce7]',
					accent: 'text-emerald-600',
					halo: 'ring-emerald-400/50 bg-emerald-50',
					icon: ShieldCheck
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

		// Background geometric circles
		ctx.fillStyle = 'rgba(255, 255, 255, 0.05)';
		ctx.beginPath();
		ctx.arc(200, 200, 260, 0, Math.PI * 2);
		ctx.fill();
		ctx.beginPath();
		ctx.arc(880, 1700, 320, 0, Math.PI * 2);
		ctx.fill();

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
		ctx.fillText('EDUPPLAY', 540, 225);

		// Tier Pill
		const pillW = 300;
		const pillH = 48;
		const pillX = 540 - pillW / 2;
		const pillY = 250;
		roundRect(ctx, pillX, pillY, pillW, pillH, 24);
		ctx.fillStyle = recap.archetype.tier === 'legendary' ? '#f59e0b' : recap.archetype.tier === 'epic' ? '#9333ea' : '#0284c7';
		ctx.fill();

		ctx.font = '800 22px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#ffffff';
		ctx.fillText(`TIER ${recap.archetype.tier_name.toUpperCase()}`, 540, pillY + 32);

		// 4. Archetype Title Banner
		ctx.font = '900 48px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#1b1d3a';
		ctx.fillText(recap.archetype.title, 540, 365);

		ctx.font = '700 24px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#5c6079';
		ctx.fillText(`${recap.archetype.tag.toUpperCase()} • ${recap.archetype.subtitle}`, 540, 410);

		// 5. Character Ring Stage
		const avY = 630;
		ctx.beginPath();
		ctx.arc(540, avY, 140, 0, Math.PI * 2);
		ctx.fillStyle = recap.archetype.tier === 'legendary' ? 'rgba(251,191,36,0.18)' : 'rgba(192,132,252,0.18)';
		ctx.fill();
		ctx.lineWidth = 6;
		ctx.strokeStyle = recap.archetype.tier === 'legendary' ? '#f59e0b' : '#9333ea';
		ctx.stroke();

		// Inner disc
		ctx.beginPath();
		ctx.arc(540, avY, 115, 0, Math.PI * 2);
		ctx.fillStyle = '#ffffff';
		ctx.fill();

		// Draw character initials in center
		ctx.font = '900 68px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#5b5ce2';
		ctx.fillText(recap.child.nickname.slice(0, 2).toUpperCase(), 540, avY + 22);

		// Level Pill under character
		const lvlW = 280;
		const lvlH = 46;
		roundRect(ctx, 540 - lvlW / 2, avY + 95, lvlW, lvlH, 23);
		ctx.fillStyle = '#1b1d3a';
		ctx.fill();
		ctx.font = '800 22px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#ffc857';
		ctx.fillText(`Level ${recap.child.level} • ${recap.child.nickname}`, 540, avY + 126);

		// 6. Stats Grid (4 Clean Boxes - No Emojis)
		const statY = 860;
		const statW = 205;
		const statH = 135;
		const statGap = 16;
		const startX = cx + 30;

		const statsList = [
			{ tag: 'STREAK', val: `${recap.stats.streak} Hari`, desc: 'Konsistensi' },
			{ tag: 'TANTANGAN', val: `${recap.stats.total_challenges}`, desc: 'Selesai' },
			{ tag: 'BINTANG', val: `${recap.stats.stars}`, desc: 'Terkumpul' },
			{ tag: 'AKURASI', val: `${Math.round(recap.stats.accuracy)}%`, desc: 'Jawaban Tepat' }
		];

		statsList.forEach((s, idx) => {
			const bx = startX + idx * (statW + statGap);
			roundRect(ctx, bx, statY, statW, statH, 20);
			ctx.fillStyle = '#ffffff';
			ctx.fill();
			ctx.lineWidth = 2.5;
			ctx.strokeStyle = '#e4e7f0';
			ctx.stroke();

			ctx.font = '800 18px "Plus Jakarta Sans", system-ui, sans-serif';
			ctx.fillStyle = '#5b5ce2';
			ctx.fillText(s.tag, bx + statW / 2, statY + 36);

			ctx.font = '900 28px "Plus Jakarta Sans", system-ui, sans-serif';
			ctx.fillStyle = '#1b1d3a';
			ctx.fillText(s.val, bx + statW / 2, statY + 80);

			ctx.font = '700 17px "Plus Jakarta Sans", system-ui, sans-serif';
			ctx.fillStyle = '#5c6079';
			ctx.fillText(s.desc, bx + statW / 2, statY + 112);
		});

		// 7. Praise Speech Bubble Card
		const bubbleY = 1050;
		const bubbleW = 860;
		const bubbleH = 440;
		const bubbleX = 540 - bubbleW / 2;

		roundRect(ctx, bubbleX, bubbleY, bubbleW, bubbleH, 32);
		ctx.fillStyle = '#ffffff';
		ctx.fill();
		ctx.lineWidth = 3;
		ctx.strokeStyle = recap.archetype.tier === 'legendary' ? '#fcd34d' : '#e4e7f0';
		ctx.stroke();

		// Header for praise box
		ctx.font = '800 22px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#5c6079';
		ctx.fillText('CATATAN PRESTASI', 540, bubbleY + 50);

		// Praise text
		ctx.font = '600 32px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#1b1d3a';
		wrapText(ctx, recap.archetype.praise, 540, bubbleY + 115, 760, 48);

		// Next tier hint inside praise card
		roundRect(ctx, bubbleX + 40, bubbleY + bubbleH - 100, bubbleW - 80, 68, 20);
		ctx.fillStyle = '#f5f6fb';
		ctx.fill();

		ctx.font = '700 22px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#5b5ce2';
		ctx.fillText(`Target Berikutnya: ${recap.archetype.next_tier_hint}`, 540, bubbleY + bubbleH - 58);

		// 8. Watermark Footer
		ctx.font = '700 24px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#8a8fa8';
		ctx.fillText('EDUPPLAY • Platform Belajar TK–SD • eduplay.id', 540, cy + ch - 50);

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
			a.download = `kartu-${recap.child.nickname.toLowerCase()}.png`;
			document.body.appendChild(a);
			a.click();
			document.body.removeChild(a);
			URL.revokeObjectURL(url);
			showToast('Gambar kartu berhasil diunduh.');
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
			const file = new File([blob], `kartu-${recap.child.nickname.toLowerCase()}.png`, { type: 'image/png' });

			if (navigator.canShare && navigator.canShare({ files: [file] })) {
				await navigator.share({
					files: [file],
					title: `Kartu Prestasi ${recap.child.nickname} di EduPlay`,
					text: `Kartu prestasi belajar ${recap.child.nickname} di EduPlay. Gelar: ${recap.archetype.title}.`
				});
				showToast('Menu bagikan berhasil dibuka.');
			} else {
				await downloadImage();
				if (navigator.clipboard) {
					await navigator.clipboard.writeText(`Kartu prestasi belajar ${recap.child.nickname} di EduPlay: ${recap.archetype.title}`);
				}
				showToast('Gambar kartu tersimpan.');
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
	<title>{recap ? `Kartu Karakter ${recap.child.nickname}` : 'Kartu Karakter'} | EduPlay</title>
</svelte:head>

<div class="relative mx-auto w-full max-w-lg px-3 py-3 sm:px-4 sm:py-6">
	<!-- Top Bar Navigation -->
	<header class="flex items-center justify-between pb-3">
		<a href="/app/play/{id}" class="inline-flex items-center gap-1.5 text-sm font-bold text-ink-soft hover:text-ink">
			<CaretLeft size={20} weight="bold" aria-hidden="true" />
			<span>Kembali</span>
		</a>
		<span class="inline-flex items-center gap-1.5 rounded-full bg-paper px-3 py-1 text-xs font-black shadow-xs border border-line">
			<Sparkle size={14} weight="fill" class="text-sun-deep" aria-hidden="true" />
			<span>Kartu Karakter</span>
		</span>
	</header>

	{#if toast}
		<div class="anim-nudge fixed top-5 left-1/2 z-50 -translate-x-1/2 rounded-full bg-night px-5 py-2.5 text-xs font-extrabold text-white shadow-lift" role="status">
			{toast}
		</div>
	{/if}

	{#if loading}
		<div class="mt-8 flex flex-col items-center justify-center gap-3 py-20 text-center">
			<div class="size-10 animate-spin rounded-full border-4 border-primary border-t-transparent"></div>
			<p class="text-sm font-bold text-ink-soft">Memuat kartu karakter…</p>
		</div>
	{:else if error || !recap}
		<div class="mt-8 rounded-2xl bg-paper p-6 text-center shadow-card">
			<p class="font-bold text-error">{error || 'Kartu belum tersedia.'}</p>
			<a href="/app/play/{id}" class="btn btn-primary mt-4">Kembali bermain</a>
		</div>
	{:else}
		<!-- Preview Stage -->
		<div class="mt-1 flex flex-col items-center">
			{#if activeSlide === 0}
				<!-- SLIDE 1: KARTU KARAKTER LEGENDARIS (BEBAS EMOJI, VEKTOR MURNI) -->
				<article
					class="relative w-full overflow-hidden rounded-[2rem] border-4 p-5 sm:p-7 {tierConfig.border} {tierConfig.cardBg} transition-all duration-300"
				>
					<!-- Subtle Background Illustration Watermark -->
					<img
						src={winnerArt}
						alt=""
						class="pointer-events-none absolute -bottom-6 -right-6 w-44 opacity-15 select-none"
					/>

					<!-- Header Brand & Tier Badge -->
					<div class="relative flex items-center justify-between border-b border-black/5 pb-3">
						<span class="font-display text-sm font-black tracking-wider text-ink">EDUPPLAY</span>
						<span class="inline-flex items-center gap-1.5 rounded-full px-3 py-0.5 text-xs font-black uppercase shadow-xs {tierConfig.badge}">
							<tierConfig.icon size={14} weight="fill" aria-hidden="true" />
							{recap.archetype.tier_name}
						</span>
					</div>

					<!-- Title & Subtitle -->
					<div class="relative mt-4 text-center">
						<h1 class="font-display text-xl font-black tracking-tight text-ink sm:text-2xl">
							{recap.archetype.title}
						</h1>
						<p class="mt-1 text-xs font-extrabold uppercase tracking-wider text-ink-soft">
							{recap.archetype.tag} • Level {recap.child.level}
						</p>
					</div>

					<!-- Character Disc Hero -->
					<div class="relative mt-5 flex justify-center">
						<div class="relative flex size-28 items-center justify-center rounded-full bg-white shadow-lift ring-6 {tierConfig.halo}">
							<Avatar code={recap.child.avatar} size={76} />
							<span class="absolute -bottom-1 -right-1 grid size-7 place-items-center rounded-full bg-night text-sun shadow-xs">
								<Sparkle size={14} weight="fill" />
							</span>
						</div>
					</div>

					<!-- Child Nickname Pill -->
					<div class="relative mt-3 text-center">
						<span class="inline-flex items-center gap-1 rounded-full bg-night px-3.5 py-1 text-xs font-black text-sun">
							<span>{recap.child.nickname}</span>
							<span class="text-white/60">•</span>
							<span>Kelas {recap.child.grade_name}</span>
						</span>
					</div>

					<!-- 4 Stat Badges (Vektor & Ikon Bersih) -->
					<div class="relative mt-5 grid grid-cols-4 gap-1.5 sm:gap-2 text-center">
						<div class="rounded-xl border border-line bg-white/95 p-2 shadow-xs">
							<span class="grid size-6 mx-auto place-items-center rounded-full bg-tangerine-tint text-tangerine-deep">
								<Fire size={14} weight="fill" />
							</span>
							<b class="mt-1 block text-sm font-black text-ink">{recap.stats.streak}</b>
							<span class="block text-[0.62rem] font-bold text-ink-soft">Streak</span>
						</div>
						<div class="rounded-xl border border-line bg-white/95 p-2 shadow-xs">
							<span class="grid size-6 mx-auto place-items-center rounded-full bg-sun-tint text-amber-700">
								<Trophy size={14} weight="fill" />
							</span>
							<b class="mt-1 block text-sm font-black text-ink">{recap.stats.total_challenges}</b>
							<span class="block text-[0.62rem] font-bold text-ink-soft">Tantangan</span>
						</div>
						<div class="rounded-xl border border-line bg-white/95 p-2 shadow-xs">
							<span class="grid size-6 mx-auto place-items-center rounded-full bg-sun-tint text-amber-600">
								<Star size={14} weight="fill" />
							</span>
							<b class="mt-1 block text-sm font-black text-ink">{recap.stats.stars}</b>
							<span class="block text-[0.62rem] font-bold text-ink-soft">Bintang</span>
						</div>
						<div class="rounded-xl border border-line bg-white/95 p-2 shadow-xs">
							<span class="grid size-6 mx-auto place-items-center rounded-full bg-primary-tint text-primary-deep">
								<Crosshair size={14} weight="fill" />
							</span>
							<b class="mt-1 block text-sm font-black text-ink">{Math.round(recap.stats.accuracy)}%</b>
							<span class="block text-[0.62rem] font-bold text-ink-soft">Akurasi</span>
						</div>
					</div>

					<!-- Praise Card Box (Kutipan Bersih) -->
					<div class="relative mt-4 rounded-2xl border border-line bg-white/95 p-3.5 shadow-xs">
						<div class="flex items-start gap-2">
							<span class="shrink-0 text-sun-deep">
								<Quotes size={20} weight="fill" />
							</span>
							<p class="text-xs font-bold leading-relaxed text-ink sm:text-sm">
								{recap.archetype.praise}
							</p>
						</div>
					</div>

					<!-- Target Milestone -->
					<div class="relative mt-3 flex items-center justify-center gap-1.5 text-center">
						<Target size={14} class="shrink-0 text-primary" weight="bold" />
						<p class="text-[0.7rem] font-extrabold text-primary-deep sm:text-xs">
							{recap.archetype.next_tier_hint}
						</p>
					</div>
				</article>
			{:else}
				<!-- SLIDE 2: RAPOR PETUALANGAN LENGKAP -->
				<article
					class="relative w-full overflow-hidden rounded-[2rem] border-4 border-line bg-white p-5 shadow-lift sm:p-7 transition-all duration-300"
				>
					<div class="flex items-center justify-between border-b border-line pb-3">
						<span class="font-display text-sm font-black text-ink">RAPOR AKTIVITAS</span>
						<span class="pill bg-sun-tint text-xs font-black text-ink">
							Level {recap.child.level} ({recap.child.xp} XP)
						</span>
					</div>

					<div class="mt-4 flex items-center gap-3">
						<Avatar code={recap.child.avatar} size={48} ring />
						<div class="min-w-0 flex-1">
							<h2 class="truncate text-lg font-black text-ink">{recap.child.nickname}</h2>
							<p class="text-xs font-semibold text-ink-soft">{recap.child.grade_name} • {recap.archetype.title}</p>
						</div>
					</div>

					<!-- 6 Tiles Breakdown -->
					<div class="mt-4 grid grid-cols-2 gap-2 text-left sm:gap-3">
						<div class="rounded-xl bg-paper-deep p-3">
							<span class="flex items-center gap-1.5 text-xs font-bold text-ink-soft">
								<Trophy size={14} class="text-sun-deep" weight="fill" />
								Tantangan
							</span>
							<p class="mt-1 text-base font-black text-ink">{recap.stats.total_challenges} <span class="text-xs font-bold text-ink-soft">selesai</span></p>
						</div>

						<div class="rounded-xl bg-paper-deep p-3">
							<span class="flex items-center gap-1.5 text-xs font-bold text-ink-soft">
								<CheckCircle size={14} class="text-leaf-deep" weight="fill" />
								Soal Selesai
							</span>
							<p class="mt-1 text-base font-black text-ink">{recap.stats.total_questions} <span class="text-xs font-bold text-ink-soft">soal</span></p>
						</div>

						<div class="rounded-xl bg-paper-deep p-3">
							<span class="flex items-center gap-1.5 text-xs font-bold text-ink-soft">
								<Crosshair size={14} class="text-primary" weight="fill" />
								Akurasi
							</span>
							<p class="mt-1 text-base font-black text-ink">{Math.round(recap.stats.accuracy)}% <span class="text-xs font-bold text-ink-soft">tepat</span></p>
						</div>

						<div class="rounded-xl bg-paper-deep p-3">
							<span class="flex items-center gap-1.5 text-xs font-bold text-ink-soft">
								<Fire size={14} class="text-tangerine" weight="fill" />
								Konsistensi
							</span>
							<p class="mt-1 text-base font-black text-ink">{recap.stats.streak} <span class="text-xs font-bold text-ink-soft">hari beruntun</span></p>
						</div>

						<div class="rounded-xl bg-paper-deep p-3">
							<span class="flex items-center gap-1.5 text-xs font-bold text-ink-soft">
								<BookOpenText size={14} class="text-rose" weight="fill" />
								Pelajaran Favorit
							</span>
							<p class="mt-1 truncate text-sm font-black text-ink">{recap.stats.favorite_subject || 'Semua Pelajaran'}</p>
						</div>

						<div class="rounded-xl bg-paper-deep p-3">
							<span class="flex items-center gap-1.5 text-xs font-bold text-ink-soft">
								<Medal size={14} class="text-sun-deep" weight="fill" />
								Lencana
							</span>
							<p class="mt-1 truncate text-sm font-black text-ink">{recap.stats.top_badge?.name || 'Penjelajah Cilik'}</p>
						</div>
					</div>

					<div class="mt-4 rounded-xl bg-primary-tint p-3 text-center">
						<p class="text-xs font-extrabold text-primary-deep">
							Terus selesaikan tantangan untuk menaikkan tier ke Legendaris.
						</p>
					</div>
				</article>
			{/if}

			<!-- Carousel Indicator -->
			<div class="mt-4 flex items-center justify-center gap-2" role="tablist" aria-label="Pilih Tampilan Kartu">
				<button
					role="tab"
					aria-selected={activeSlide === 0}
					aria-label="Tampilan Kartu Karakter"
					onclick={() => (activeSlide = 0)}
					class="transition-all duration-300 {activeSlide === 0 ? 'h-2 w-7 rounded-full bg-sun' : 'size-2 rounded-full bg-line hover:bg-ink-soft'}"
				></button>
				<button
					role="tab"
					aria-selected={activeSlide === 1}
					aria-label="Tampilan Rapor Aktivitas"
					onclick={() => (activeSlide = 1)}
					class="transition-all duration-300 {activeSlide === 1 ? 'h-2 w-7 rounded-full bg-sun' : 'size-2 rounded-full bg-line hover:bg-ink-soft'}"
				></button>
			</div>

			<!-- Action Buttons (Mobile-first, touch-friendly) -->
			<div class="mt-4 flex w-full flex-col gap-2.5">
				<button
					class="btn btn-sun w-full shadow-card py-3 text-sm font-extrabold"
					onclick={shareToInstagram}
					disabled={exporting}
				>
					<InstagramLogo size={20} weight="fill" aria-hidden="true" />
					<span>{exporting ? 'Menyiapkan Gambar…' : 'Bagikan di Instagram / Medsos'}</span>
				</button>

				<button
					class="inline-flex items-center justify-center gap-2 py-2.5 text-xs font-bold text-ink-soft hover:text-ink transition"
					onclick={downloadImage}
					disabled={exporting}
				>
					<DownloadSimple size={16} weight="bold" aria-hidden="true" />
					<span>Unduh Gambar Kartu (PNG)</span>
				</button>
			</div>
		</div>
	{/if}
</div>
