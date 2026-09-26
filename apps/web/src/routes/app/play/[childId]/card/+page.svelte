<script lang="ts">
	import { get } from '$lib/api';
	import { page } from '$app/state';
	import type { ChildRecap } from '$lib/types';
	import LegendaryCardFrame from '$lib/components/LegendaryCardFrame.svelte';
	import CaretLeft from 'phosphor-svelte/lib/CaretLeft';
	import Sparkle from 'phosphor-svelte/lib/Sparkle';
	import Star from 'phosphor-svelte/lib/Star';
	import Fire from 'phosphor-svelte/lib/Fire';
	import Trophy from 'phosphor-svelte/lib/Trophy';
	import Crosshair from 'phosphor-svelte/lib/Crosshair';
	import Medal from 'phosphor-svelte/lib/Medal';
	import BookOpenText from 'phosphor-svelte/lib/BookOpenText';
	import InstagramLogo from 'phosphor-svelte/lib/InstagramLogo';
	import DownloadSimple from 'phosphor-svelte/lib/DownloadSimple';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	import Avatar from '$lib/components/Avatar.svelte';

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

	// High fidelity 1080x1920 Instagram Story Canvas generator matching the reference layout
	async function generateCanvas(): Promise<Blob | null> {
		if (!recap) return null;
		const canvas = document.createElement('canvas');
		canvas.width = 1080;
		canvas.height = 1920;
		const ctx = canvas.getContext('2d');
		if (!ctx) return null;

		// 1. Dark outer cosmic stage background
		const bg = ctx.createLinearGradient(0, 0, 1080, 1920);
		bg.addColorStop(0, '#0a0d24');
		bg.addColorStop(0.5, '#15193d');
		bg.addColorStop(1, '#080a1c');
		ctx.fillStyle = bg;
		ctx.fillRect(0, 0, 1080, 1920);

		// Subtle celestial rays
		ctx.fillStyle = 'rgba(255, 255, 255, 0.04)';
		for (let i = 0; i < 8; i++) {
			ctx.beginPath();
			ctx.moveTo(540, 400);
			ctx.lineTo(i * 180, 0);
			ctx.lineTo((i + 1) * 180, 0);
			ctx.closePath();
			ctx.fill();
		}

		// 2. Outer Ornate Golden Border Frame
		const ox = 50;
		const oy = 60;
		const ow = 980;
		const oh = 1800;
		const or = 54;

		// Outer Gold Gradient Bevel
		const goldGrad = ctx.createLinearGradient(ox, oy, ox + ow, oy + oh);
		goldGrad.addColorStop(0, '#dfad41');
		goldGrad.addColorStop(0.3, '#ffd66c');
		goldGrad.addColorStop(0.7, '#f59e0b');
		goldGrad.addColorStop(1, '#8c5311');

		roundRect(ctx, ox, oy, ow, oh, or);
		ctx.fillStyle = goldGrad;
		ctx.shadowColor = 'rgba(0, 0, 0, 0.6)';
		ctx.shadowBlur = 40;
		ctx.fill();

		// Inner White Inset Card Stage
		const ix = ox + 18;
		const iy = oy + 18;
		const iw = ow - 36;
		const ih = oh - 36;
		const ir = 44;

		ctx.save();
		roundRect(ctx, ix, iy, iw, ih, ir);
		ctx.clip();

		const cardBg = ctx.createLinearGradient(ix, iy, ix, iy + ih);
		cardBg.addColorStop(0, '#f8faff');
		cardBg.addColorStop(0.4, '#ffffff');
		cardBg.addColorStop(1, '#f4f1e8');
		ctx.fillStyle = cardBg;
		ctx.fillRect(ix, iy, iw, ih);

		// 3. Header Bar: EduPlay Wordmark + Date Pill
		ctx.textAlign = 'left';
		ctx.font = '900 36px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#3f40c2';
		ctx.fillText('EduPlay', ix + 36, iy + 64);

		ctx.font = '700 20px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#5c6079';
		ctx.fillText('Belajar Jadi Lebih Seru', ix + 36, iy + 92);

		// Top Right Date Tag
		const dateW = 200;
		const dateH = 44;
		const dateX = ix + iw - dateW - 36;
		const dateY = iy + 45;
		roundRect(ctx, dateX, dateY, dateW, dateH, 22);
		ctx.fillStyle = '#ffffff';
		ctx.fill();
		ctx.lineWidth = 2;
		ctx.strokeStyle = '#fcd34d';
		ctx.stroke();

		ctx.textAlign = 'center';
		ctx.font = '800 20px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#1b1d3a';
		ctx.fillText(recap.archetype.date_formatted, dateX + dateW / 2, dateY + 29);

		// Motto Watermark
		ctx.textAlign = 'left';
		ctx.font = '800 22px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#78716c';
		ctx.fillText(`"${recap.archetype.motto}"`, ix + 36, iy + 140);

		// 4. Hero Character Stage: Sky, Castle silhouette, Golden stars
		const heroY = iy + 160;
		const heroH = 500;
		const heroGrad = ctx.createLinearGradient(ix, heroY, ix, heroY + heroH);
		heroGrad.addColorStop(0, '#38bdf8');
		heroGrad.addColorStop(0.6, '#93c5fd');
		heroGrad.addColorStop(1, '#fef08a');
		ctx.fillStyle = heroGrad;
		ctx.fillRect(ix, heroY, iw, heroH);

		// Floating Stars
		ctx.fillStyle = '#ffeb3b';
		for (let s = 0; s < 12; s++) {
			const sx = ix + 60 + ((s * 179) % (iw - 120));
			const sy = heroY + 30 + ((s * 113) % 240);
			ctx.beginPath();
			ctx.arc(sx, sy, 8, 0, Math.PI * 2);
			ctx.fill();
		}

		// Hero Center Circle & Glow
		const hcx = ix + iw / 2;
		const hcy = heroY + 230;

		// Sunburst Glow behind hero
		ctx.fillStyle = 'rgba(255, 255, 255, 0.4)';
		ctx.beginPath();
		ctx.arc(hcx, hcy, 180, 0, Math.PI * 2);
		ctx.fill();

		// Hero Disc
		ctx.fillStyle = '#ffffff';
		ctx.beginPath();
		ctx.arc(hcx, hcy, 120, 0, Math.PI * 2);
		ctx.fill();
		ctx.lineWidth = 12;
		ctx.strokeStyle = '#fcd34d';
		ctx.stroke();

		// Character Initials / Avatar representation
		ctx.textAlign = 'center';
		ctx.font = '900 80px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#3f40c2';
		ctx.fillText(recap.child.nickname.slice(0, 2).toUpperCase(), hcx, hcy + 28);

		// Companion Robot (Right side)
		const rcx = hcx + 230;
		const rcy = heroY + 140;
		roundRect(ctx, rcx - 45, rcy - 35, 90, 70, 24);
		ctx.fillStyle = '#ffffff';
		ctx.fill();
		ctx.lineWidth = 4;
		ctx.strokeStyle = '#38bdf8';
		ctx.stroke();

		// Robot Eyes
		roundRect(ctx, rcx - 30, rcy - 16, 60, 32, 12);
		ctx.fillStyle = '#1b1d3a';
		ctx.fill();
		ctx.fillStyle = '#38bdf8';
		ctx.beginPath();
		ctx.arc(rcx - 14, rcy, 6, 0, Math.PI * 2);
		ctx.arc(rcx + 14, rcy, 6, 0, Math.PI * 2);
		ctx.fill();

		// Robot Crown
		ctx.fillStyle = '#fbbf24';
		ctx.beginPath();
		ctx.moveTo(rcx - 24, rcy - 36);
		ctx.lineTo(rcx - 24, rcy - 52);
		ctx.lineTo(rcx - 12, rcy - 44);
		ctx.lineTo(rcx, rcy - 56);
		ctx.lineTo(rcx + 12, rcy - 44);
		ctx.lineTo(rcx + 24, rcy - 52);
		ctx.lineTo(rcx + 24, rcy - 36);
		ctx.closePath();
		ctx.fill();

		// 5. Huge 3D Golden Banner: LEGENDARY / LEARNER
		const bannerY = heroY + heroH - 60;

		// Golden Star on Crest
		ctx.beginPath();
		ctx.arc(hcx, bannerY - 30, 32, 0, Math.PI * 2);
		ctx.fillStyle = '#f59e0b';
		ctx.fill();
		ctx.lineWidth = 4;
		ctx.strokeStyle = '#ffffff';
		ctx.stroke();

		// Main Banner Ribbon
		const bw = 540;
		const bh = 82;
		const bx = hcx - bw / 2;
		roundRect(ctx, bx, bannerY, bw, bh, 24);
		ctx.fillStyle = '#581c87';
		ctx.fill();
		ctx.lineWidth = 6;
		ctx.strokeStyle = '#fcd34d';
		ctx.stroke();

		ctx.textAlign = 'center';
		ctx.font = '900 44px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#fef08a';
		ctx.fillText(recap.archetype.banner, hcx, bannerY + 56);

		// Sub Ribbon (e.g. LEARNER / KSATRIA API)
		const sbw = 340;
		const sbh = 44;
		const sbx = hcx - sbw / 2;
		const sby = bannerY + bh - 14;
		roundRect(ctx, sbx, sby, sbw, sbh, 16);
		ctx.fillStyle = '#f59e0b';
		ctx.fill();
		ctx.lineWidth = 3;
		ctx.strokeStyle = '#78350f';
		ctx.stroke();

		ctx.font = '900 24px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#1b1d3a';
		ctx.fillText(recap.archetype.sub_banner, hcx, sby + 31);

		// 6. Parchment Profile Strip (Level Shield, Nickname, Quote)
		const profY = sby + 65;
		const profW = iw - 60;
		const profH = 120;
		const profX = ix + 30;

		roundRect(ctx, profX, profY, profW, profH, 24);
		ctx.fillStyle = '#fdfbf2';
		ctx.fill();
		ctx.lineWidth = 3;
		ctx.strokeStyle = '#fcd34d';
		ctx.stroke();

		// Level Shield
		roundRect(ctx, profX + 20, profY + 16, 75, 88, 16);
		ctx.fillStyle = '#92400e';
		ctx.fill();
		ctx.lineWidth = 2;
		ctx.strokeStyle = '#fde68a';
		ctx.stroke();

		ctx.font = '900 16px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#fde68a';
		ctx.fillText('LEVEL', profX + 57, profY + 42);

		ctx.font = '900 36px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#ffffff';
		ctx.fillText(String(recap.child.level), profX + 57, profY + 84);

		// Nickname
		ctx.textAlign = 'left';
		ctx.font = '900 38px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#1b1d3a';
		ctx.fillText(recap.child.nickname, profX + 115, profY + 56);

		ctx.font = '700 22px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#78716c';
		ctx.fillText(recap.archetype.title, profX + 115, profY + 92);

		// 7. 4 Stat Boxes (Bintang, Streak, Akurasi, Soal)
		const statY = profY + profH + 25;
		const statBoxW = (profW - 36) / 4;
		const statBoxH = 190;

		const stats = [
			{ icon: '⭐', label: 'Bintang', val: String(recap.stats.stars), up: '+120' },
			{ icon: '🔥', label: 'Streak Hari', val: `${recap.stats.streak}`, up: '+5' },
			{ icon: '🎯', label: 'Akurasi', val: `${Math.round(recap.stats.accuracy)}%`, up: '+8%' },
			{ icon: '📖', label: 'Soal', val: String(recap.stats.total_questions), up: '+24' }
		];

		stats.forEach((s, idx) => {
			const sx = profX + idx * (statBoxW + 12);
			roundRect(ctx, sx, statY, statBoxW, statBoxH, 20);
			ctx.fillStyle = '#ffffff';
			ctx.fill();
			ctx.lineWidth = 2.5;
			ctx.strokeStyle = '#fcd34d';
			ctx.stroke();

			ctx.textAlign = 'center';
			ctx.font = '32px sans-serif';
			ctx.fillText(s.icon, sx + statBoxW / 2, statY + 52);

			ctx.font = '900 34px "Plus Jakarta Sans", system-ui, sans-serif';
			ctx.fillStyle = '#1b1d3a';
			ctx.fillText(s.val, sx + statBoxW / 2, statY + 104);

			ctx.font = '700 20px "Plus Jakarta Sans", system-ui, sans-serif';
			ctx.fillStyle = '#78716c';
			ctx.fillText(s.label, sx + statBoxW / 2, statY + 138);

			// Upward Badge
			roundRect(ctx, sx + statBoxW / 2 - 38, statY + 148, 76, 30, 12);
			ctx.fillStyle = '#e4f7ec';
			ctx.fill();
			ctx.font = '800 18px "Plus Jakarta Sans", system-ui, sans-serif';
			ctx.fillStyle = '#1f7a4d';
			ctx.fillText(s.up, sx + statBoxW / 2, statY + 170);
		});

		// 8. Bottom Row: Progress to Next Tier & Social Proof Trophy
		const btmY = statY + statBoxH + 25;
		const btmW = (profW - 16) / 2;
		const btmH = 135;

		// Left Card: Next Tier
		roundRect(ctx, profX, btmY, btmW, btmH, 22);
		ctx.fillStyle = '#ffffff';
		ctx.fill();
		ctx.lineWidth = 2.5;
		ctx.strokeStyle = '#fcd34d';
		ctx.stroke();

		ctx.textAlign = 'left';
		ctx.font = '700 19px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#78716c';
		ctx.fillText('Menuju Tier Berikutnya', profX + 24, btmY + 40);

		ctx.font = '900 28px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#1b1d3a';
		ctx.fillText(recap.archetype.next_tier_name, profX + 24, btmY + 76);

		// Progress Bar
		roundRect(ctx, profX + 24, btmY + 92, btmW - 48, 18, 9);
		ctx.fillStyle = '#eef0f7';
		ctx.fill();
		roundRect(ctx, profX + 24, btmY + 92, (btmW - 48) * (recap.archetype.progress_percent / 100), 18, 9);
		ctx.fillStyle = '#7c3aed';
		ctx.fill();

		// Right Card: Social Proof
		roundRect(ctx, profX + btmW + 16, btmY, btmW, btmH, 22);
		ctx.fillStyle = '#ffffff';
		ctx.fill();
		ctx.lineWidth = 2.5;
		ctx.strokeStyle = '#fcd34d';
		ctx.stroke();

		ctx.textAlign = 'left';
		ctx.font = '900 24px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#b45309';
		ctx.fillText('Luar biasa!', profX + btmW + 40, btmY + 44);

		ctx.font = '700 19px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#5c6079';
		wrapText(
			ctx,
			`Kamu sudah lebih baik dari ${recap.archetype.peer_percentile}% teman-teman di EduPlay!`,
			profX + btmW + 40,
			btmY + 74,
			btmW - 60,
			28
		);

		// 9. Bottom Footer Bar with Feature Icons
		const footY = iy + ih - 90;
		roundRect(ctx, profX, footY, profW, 64, 18);
		ctx.fillStyle = '#191a38';
		ctx.fill();

		ctx.textAlign = 'center';
		ctx.font = '800 20px "Plus Jakarta Sans", system-ui, sans-serif';
		ctx.fillStyle = '#fcd34d';
		ctx.fillText('🎓 Belajar Mudah   |   🎮 Tantangan Seru   |   📊 Perkembangan   |   👥 Bersama Teman', hcx, footY + 40);

		ctx.restore();

		// Diamond Gemstones at 4 corners
		const gemSize = 44;
		const gems = [
			{ x: ox, y: oy },
			{ x: ox + ow, y: oy },
			{ x: ox, y: oy + oh },
			{ x: ox + ow, y: oy + oh }
		];
		gems.forEach((g) => {
			ctx.save();
			ctx.translate(g.x, g.y);
			ctx.rotate(Math.PI / 4);
			roundRect(ctx, -gemSize / 2, -gemSize / 2, gemSize, gemSize, 6);
			ctx.fillStyle = recap?.archetype.code === 'ksatria_api' ? '#ef4444' : '#0284c7';
			ctx.fill();
			ctx.lineWidth = 4;
			ctx.strokeStyle = '#ffffff';
			ctx.stroke();
			ctx.restore();
		});

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
			showToast('Gambar kartu HD berhasil diunduh.');
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

<div class="relative mx-auto w-full max-w-md px-2 py-2 sm:max-w-lg sm:px-4 sm:py-5">
	<!-- Top Bar Navigation -->
	<header class="flex items-center justify-between pb-2 sm:pb-3">
		<a href="/app/play/{id}" class="inline-flex items-center gap-1.5 text-sm font-bold text-ink-soft hover:text-ink">
			<CaretLeft size={20} weight="bold" aria-hidden="true" />
			<span>Kembali</span>
		</a>
		<span class="hidden sm:inline-flex items-center gap-1.5 rounded-full bg-paper px-3 py-1 text-xs font-black shadow-xs border border-line">
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
		<div class="mt-8 flex flex-col items-center justify-center gap-3 py-24 text-center">
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
				<!-- SLIDE 1: KARTU TCG FANTASY MEWAH SESUAI REFERENSI -->
				<LegendaryCardFrame {recap} />
			{:else}
				<!-- SLIDE 2: RAPOR AKTIVITAS DETAIL -->
				<article
					class="relative w-full overflow-hidden rounded-[2.2rem] border-4 border-line bg-white p-5 shadow-lift sm:p-7 transition-all duration-300"
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
					<span>{exporting ? 'Menyiapkan Gambar HD…' : 'Bagikan di Instagram / Story'}</span>
				</button>

				<button
					class="inline-flex items-center justify-center gap-2 py-2.5 text-xs font-bold text-ink-soft hover:text-ink transition"
					onclick={downloadImage}
					disabled={exporting}
				>
					<DownloadSimple size={16} weight="bold" aria-hidden="true" />
					<span>Unduh Gambar Kartu HD (PNG)</span>
				</button>
			</div>
		</div>
	{/if}
</div>
