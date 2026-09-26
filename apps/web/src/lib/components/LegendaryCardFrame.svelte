<script lang="ts">
	import type { ChildRecap } from '$lib/types';
	import LegendaryHeroScene from './LegendaryHeroScene.svelte';
	import Star from 'phosphor-svelte/lib/Star';
	import Fire from 'phosphor-svelte/lib/Fire';
	import Target from 'phosphor-svelte/lib/Target';
	import BookOpen from 'phosphor-svelte/lib/BookOpen';
	import Trophy from 'phosphor-svelte/lib/Trophy';
	import Sparkle from 'phosphor-svelte/lib/Sparkle';
	import CalendarBlank from 'phosphor-svelte/lib/CalendarBlank';
	import GraduationCap from 'phosphor-svelte/lib/GraduationCap';
	import GameController from 'phosphor-svelte/lib/GameController';
	import ChartBar from 'phosphor-svelte/lib/ChartBar';
	import UsersThree from 'phosphor-svelte/lib/UsersThree';
	import PenNib from 'phosphor-svelte/lib/PenNib';

	let { recap }: { recap: ChildRecap } = $props();

	// Color theme and wing styles tailored to each archetype
	const theme = $derived.by(() => {
		switch (recap.archetype.code) {
			case 'ksatria_api':
				return {
					gem: '#ef4444',
					gemBorder: '#f87171',
					bannerGrad: 'from-amber-600 via-rose-700 to-amber-700',
					subRibbonGrad: 'from-yellow-300 via-amber-400 to-yellow-500',
					wingGrad: 'from-amber-300 via-yellow-400 to-orange-500',
					haloRing: '#f59e0b',
					bgTint: '#fffaf5'
				};
			case 'detektif_elang':
				return {
					gem: '#0284c7',
					gemBorder: '#38bdf8',
					bannerGrad: 'from-blue-700 via-indigo-800 to-cyan-800',
					subRibbonGrad: 'from-sky-300 via-blue-400 to-sky-500',
					wingGrad: 'from-sky-300 via-cyan-400 to-blue-500',
					haloRing: '#0284c7',
					bgTint: '#f0f9ff'
				};
			case 'penyihir_angka':
				return {
					gem: '#a855f7',
					gemBorder: '#c084fc',
					bannerGrad: 'from-purple-800 via-fuchsia-900 to-indigo-900',
					subRibbonGrad: 'from-amber-300 via-yellow-400 to-amber-500',
					wingGrad: 'from-purple-300 via-fuchsia-400 to-indigo-500',
					haloRing: '#9333ea',
					bgTint: '#faf5ff'
				};
			case 'penjelajah_kata':
				return {
					gem: '#10b981',
					gemBorder: '#34d399',
					bannerGrad: 'from-emerald-800 via-teal-900 to-emerald-900',
					subRibbonGrad: 'from-yellow-300 via-amber-400 to-yellow-500',
					wingGrad: 'from-emerald-300 via-teal-400 to-emerald-500',
					haloRing: '#059669',
					bgTint: '#f0fdf4'
				};
			case 'kolektor_bintang':
				return {
					gem: '#f59e0b',
					gemBorder: '#fde047',
					bannerGrad: 'from-amber-600 via-yellow-700 to-amber-800',
					subRibbonGrad: 'from-yellow-200 via-amber-300 to-yellow-400',
					wingGrad: 'from-yellow-200 via-amber-400 to-yellow-500',
					haloRing: '#f59e0b',
					bgTint: '#fffdf5'
				};
			default:
				return {
					gem: '#3b82f6',
					gemBorder: '#93c5fd',
					bannerGrad: 'from-indigo-700 via-purple-800 to-blue-800',
					subRibbonGrad: 'from-yellow-300 via-amber-400 to-yellow-500',
					wingGrad: 'from-amber-300 via-yellow-400 to-amber-500',
					haloRing: '#6366f1',
					bgTint: '#fafaff'
				};
		}
	});
</script>

<!-- The Grand Fantasy TCG Card Container -->
<div class="relative w-full overflow-hidden rounded-[2.6rem] p-3 sm:p-4 bg-gradient-to-b from-[#dfad41] via-[#ffd66c] to-[#976214] shadow-[0_20px_60px_rgba(0,0,0,0.35)] select-none">
	<!-- Outer Metallic Bevel Inset -->
	<div class="relative w-full overflow-hidden rounded-[2.2rem] border-2 border-amber-200/90 bg-[#fdfbf4] p-2.5 sm:p-3.5 shadow-inner">
		<!-- 4 Corner Faceted Gemstones -->
		<!-- Top Left Gem -->
		<div class="absolute -top-1 -left-1 z-30 size-7 sm:size-8 rotate-45 border-2 border-white shadow-md" style="background: {theme.gem};"></div>
		<!-- Top Right Gem -->
		<div class="absolute -top-1 -right-1 z-30 size-7 sm:size-8 rotate-45 border-2 border-white shadow-md" style="background: {theme.gem};"></div>
		<!-- Bottom Left Gem -->
		<div class="absolute -bottom-1 -left-1 z-30 size-7 sm:size-8 rotate-45 border-2 border-white shadow-md" style="background: {theme.gem};"></div>
		<!-- Bottom Right Gem -->
		<div class="absolute -bottom-1 -right-1 z-30 size-7 sm:size-8 rotate-45 border-2 border-white shadow-md" style="background: {theme.gem};"></div>

		<!-- Card Content Stage -->
		<div class="relative flex flex-col overflow-hidden rounded-[1.8rem] bg-gradient-to-b from-[#f8f9ff] via-[#ffffff] to-[#f4f1e8]">
			<!-- 1. Header Bar: EduPlay Wordmark + Date Tag -->
			<div class="relative z-20 flex items-center justify-between px-3.5 pt-3 pb-1">
				<div class="flex items-center gap-1.5">
					<div class="flex size-7 items-center justify-center rounded-lg bg-gradient-to-tr from-amber-500 to-yellow-300 text-ink shadow-xs">
						<Sparkle size={16} weight="fill" />
					</div>
					<div>
						<span class="font-display text-sm font-black tracking-wide text-primary-deep">EduPlay</span>
						<span class="block text-[0.6rem] font-bold text-ink-soft">Belajar Jadi Lebih Seru</span>
					</div>
				</div>

				<div class="flex items-center gap-1 rounded-full border border-amber-300/80 bg-white/90 px-2.5 py-1 text-[0.68rem] font-black text-ink shadow-2xs">
					<CalendarBlank size={12} weight="bold" class="text-amber-600" />
					<span>{recap.archetype.date_formatted}</span>
				</div>
			</div>

			<!-- Dynamic Motto / Motivational text top left watermark -->
			<div class="relative z-20 px-4 pt-1 pb-1">
				<p class="max-w-[240px] text-[0.62rem] font-extrabold leading-tight text-ink/70">
					"{recap.archetype.motto}"
				</p>
			</div>

			<!-- 2. Hero Illustration Canvas Stage -->
			<div class="relative">
				<LegendaryHeroScene child={recap.child} archetype={recap.archetype} />

				<!-- 3. Giant 3D Golden Crest & Wings Ribbon Overlay -->
				<div class="relative -mt-12 sm:-mt-14 z-20 flex flex-col items-center px-2">
					<!-- Top Golden Star Crown -->
					<div class="flex size-10 items-center justify-center rounded-full bg-gradient-to-tr from-yellow-500 via-amber-300 to-yellow-100 text-amber-900 shadow-lg ring-2 ring-white">
						<Star size={20} weight="fill" />
					</div>

					<!-- Majestic Multi-layered Wings & Banner -->
					<div class="relative -mt-2 flex w-full max-w-sm items-center justify-center">
						<!-- Left Wings SVG -->
						<svg viewBox="0 0 100 48" class="h-10 sm:h-12 w-24 shrink-0 drop-shadow">
							<path d="M 100 24 Q 60 5 10 0 Q 30 18 5 18 Q 35 30 15 36 Q 50 42 100 24 Z" fill="url(#goldRibbon)" stroke="#b45309" stroke-width="1.5" />
						</svg>

						<!-- Central 3D Curved Crest Banner -->
						<div class="relative -mx-3 flex flex-col items-center">
							<!-- Primary Tier Banner (e.g. LEGENDARY) -->
							<div class="relative flex items-center justify-center rounded-xl bg-gradient-to-r {theme.bannerGrad} px-5 sm:px-7 py-1.5 shadow-[0_8px_20px_rgba(0,0,0,0.3)] ring-3 ring-amber-300">
								<span class="font-display text-base sm:text-xl font-black tracking-widest text-amber-100 drop-shadow-[0_2px_4px_rgba(0,0,0,0.8)]">
									{recap.archetype.banner}
								</span>
							</div>

							<!-- Sub-Ribbon (e.g. LEARNER / KSATRIA API) -->
							<div class="relative -mt-1.5 flex items-center justify-center rounded-lg bg-gradient-to-r {theme.subRibbonGrad} px-3.5 sm:px-5 py-0.5 shadow-md border border-amber-600">
								<span class="text-[0.65rem] sm:text-xs font-black tracking-wider text-ink uppercase">
									{recap.archetype.sub_banner}
								</span>
							</div>
						</div>

						<!-- Right Wings SVG -->
						<svg viewBox="0 0 100 48" class="h-10 sm:h-12 w-24 shrink-0 drop-shadow -scale-x-100">
							<path d="M 100 24 Q 60 5 10 0 Q 30 18 5 18 Q 35 30 15 36 Q 50 42 100 24 Z" fill="url(#goldRibbon)" stroke="#b45309" stroke-width="1.5" />
						</svg>
					</div>
				</div>
			</div>

			<!-- 4. Parchment Profile Strip (Level Shield, Nickname, Quote) -->
			<div class="relative z-10 mx-2 sm:mx-3.5 mt-3 rounded-2xl border-2 border-amber-300/80 bg-gradient-to-b from-[#fffcf2] to-[#fbf6e6] p-2.5 sm:p-3 shadow-xs">
				<div class="grid grid-cols-[auto_1fr_auto] items-center gap-2 sm:gap-3">
					<!-- Shield Level Crest -->
					<div class="flex flex-col items-center justify-center rounded-xl bg-gradient-to-b from-amber-600 to-amber-900 px-2.5 py-1.5 text-white shadow-sm ring-2 ring-amber-300">
						<span class="text-[0.6rem] font-black uppercase tracking-wider text-amber-200">Level</span>
						<span class="font-display text-base sm:text-lg font-black leading-none">{recap.child.level}</span>
					</div>

					<!-- Nickname Plate -->
					<div class="min-w-0 text-center">
						<h2 class="truncate font-display text-base sm:text-lg font-black text-ink">{recap.child.nickname}</h2>
						<span class="inline-flex items-center gap-1 text-[0.65rem] font-bold text-ink-soft">
							<PenNib size={11} class="text-amber-700" />
							<span>{recap.archetype.title}</span>
						</span>
					</div>

					<!-- Speech / Note Quote Card -->
					<div class="hidden sm:block max-w-[150px] rounded-xl border border-amber-200 bg-white/95 p-2 text-right shadow-2xs">
						<p class="text-[0.6rem] font-bold italic leading-tight text-ink-soft">
							"{recap.archetype.quote}"
						</p>
					</div>
				</div>
			</div>

			<!-- 5. 4 Stat Tiles Grid (3D icons + Value + Green Increase Tag) -->
			<div class="relative z-10 mx-2 sm:mx-3.5 mt-2.5 grid grid-cols-4 gap-1.5 sm:gap-2 text-center">
				<!-- 1. Bintang -->
				<div class="flex flex-col items-center justify-between rounded-xl border border-amber-200/90 bg-white p-2 shadow-xs">
					<div class="flex size-7 items-center justify-center rounded-full bg-amber-100 text-amber-500 shadow-2xs">
						<Star size={16} weight="fill" />
					</div>
					<div class="mt-1">
						<b class="font-display block text-sm sm:text-base font-black text-ink">{recap.stats.stars}</b>
						<span class="text-[0.6rem] font-extrabold text-ink-soft">Bintang</span>
					</div>
					<span class="mt-1 inline-flex items-center rounded-full bg-leaf-tint px-1.5 py-0.2 text-[0.55rem] font-black text-leaf-deep">
						+{recap.stats.stars > 10 ? recap.stats.stars * 5 : 120}
					</span>
				</div>

				<!-- 2. Streak Hari -->
				<div class="flex flex-col items-center justify-between rounded-xl border border-amber-200/90 bg-white p-2 shadow-xs">
					<div class="flex size-7 items-center justify-center rounded-full bg-tangerine-tint text-tangerine-deep shadow-2xs">
						<Fire size={16} weight="fill" />
					</div>
					<div class="mt-1">
						<b class="font-display block text-sm sm:text-base font-black text-ink">{recap.stats.streak}</b>
						<span class="text-[0.6rem] font-extrabold text-ink-soft">Streak Hari</span>
					</div>
					<span class="mt-1 inline-flex items-center rounded-full bg-leaf-tint px-1.5 py-0.2 text-[0.55rem] font-black text-leaf-deep">
						+{recap.stats.streak > 0 ? recap.stats.streak : 1}
					</span>
				</div>

				<!-- 3. Akurasi -->
				<div class="flex flex-col items-center justify-between rounded-xl border border-amber-200/90 bg-white p-2 shadow-xs">
					<div class="flex size-7 items-center justify-center rounded-full bg-sky-tint text-sky shadow-2xs">
						<Target size={16} weight="fill" />
					</div>
					<div class="mt-1">
						<b class="font-display block text-sm sm:text-base font-black text-ink">{Math.round(recap.stats.accuracy)}%</b>
						<span class="text-[0.6rem] font-extrabold text-ink-soft">Akurasi</span>
					</div>
					<span class="mt-1 inline-flex items-center rounded-full bg-leaf-tint px-1.5 py-0.2 text-[0.55rem] font-black text-leaf-deep">
						+8%
					</span>
				</div>

				<!-- 4. Soal Selesai -->
				<div class="flex flex-col items-center justify-between rounded-xl border border-amber-200/90 bg-white p-2 shadow-xs">
					<div class="flex size-7 items-center justify-center rounded-full bg-primary-tint text-primary-deep shadow-2xs">
						<BookOpen size={16} weight="fill" />
					</div>
					<div class="mt-1">
						<b class="font-display block text-sm sm:text-base font-black text-ink">{recap.stats.total_questions}</b>
						<span class="text-[0.6rem] font-extrabold text-ink-soft">Soal</span>
					</div>
					<span class="mt-1 inline-flex items-center rounded-full bg-leaf-tint px-1.5 py-0.2 text-[0.55rem] font-black text-leaf-deep">
						+{recap.stats.total_challenges}
					</span>
				</div>
			</div>

			<!-- 6. Next Tier Progress & Social Proof Row -->
			<div class="relative z-10 mx-2 sm:mx-3.5 mt-2.5 grid grid-cols-1 sm:grid-cols-2 gap-2 text-left">
				<!-- Tier Progress Bar Card -->
				<div class="flex items-center gap-2.5 rounded-xl border border-amber-200/90 bg-white p-2.5 shadow-xs">
					<!-- Gemstone Tier Crest -->
					<div class="flex size-9 shrink-0 items-center justify-center rounded-xl bg-gradient-to-tr from-purple-700 to-indigo-500 text-white shadow-sm ring-2 ring-purple-300">
						<Sparkle size={18} weight="fill" />
					</div>
					<div class="min-w-0 flex-1">
						<div class="flex items-center justify-between text-[0.62rem] font-extrabold text-ink-soft">
							<span>Menuju Tier Berikutnya</span>
							<span class="text-primary font-black">{recap.archetype.progress_percent}%</span>
						</div>
						<b class="block text-xs font-black text-ink truncate">{recap.archetype.next_tier_name}</b>
						<!-- Progress track -->
						<div class="mt-1 h-2 w-full overflow-hidden rounded-full bg-paper-deep">
							<div
								class="h-full rounded-full bg-gradient-to-r from-purple-600 to-primary transition-all duration-500"
								style="width: {recap.archetype.progress_percent}%;"
							></div>
						</div>
					</div>
				</div>

				<!-- Social Proof Trophy Card -->
				<div class="flex items-center gap-2.5 rounded-xl border border-amber-200/90 bg-white p-2.5 shadow-xs">
					<div class="flex size-9 shrink-0 items-center justify-center rounded-xl bg-gradient-to-tr from-amber-400 to-yellow-200 text-amber-800 shadow-sm ring-2 ring-amber-300">
						<Trophy size={18} weight="fill" />
					</div>
					<div class="min-w-0 flex-1">
						<b class="block text-xs font-black text-amber-700">Luar biasa!</b>
						<p class="text-[0.6rem] font-bold leading-tight text-ink-soft">
							Kamu sudah lebih baik dari {recap.archetype.peer_percentile}% teman-teman di EduPlay!
						</p>
					</div>
				</div>
			</div>

			<!-- 7. Bottom Feature Bar with Gold Dividers -->
			<div class="relative z-10 mx-2 sm:mx-3.5 my-2.5 flex items-center justify-around rounded-xl bg-gradient-to-r from-[#191a38] via-[#212352] to-[#191a38] px-2 py-2 text-white/90 shadow-sm">
				<div class="flex items-center gap-1.5 text-[0.58rem] sm:text-[0.65rem] font-extrabold">
					<GraduationCap size={14} class="text-amber-400" />
					<span>Belajar Mudah</span>
				</div>
				<span class="text-white/20">|</span>
				<div class="flex items-center gap-1.5 text-[0.58rem] sm:text-[0.65rem] font-extrabold">
					<GameController size={14} class="text-amber-400" />
					<span>Tantangan Seru</span>
				</div>
				<span class="text-white/20">|</span>
				<div class="flex items-center gap-1.5 text-[0.58rem] sm:text-[0.65rem] font-extrabold">
					<ChartBar size={14} class="text-amber-400" />
					<span>Perkembangan</span>
				</div>
				<span class="text-white/20">|</span>
				<div class="flex items-center gap-1.5 text-[0.58rem] sm:text-[0.65rem] font-extrabold">
					<UsersThree size={14} class="text-amber-400" />
					<span>Bersama Teman</span>
				</div>
			</div>
		</div>

		<!-- Centered Bottom Gemstone Anchor -->
		<div class="absolute -bottom-2 left-1/2 -translate-x-1/2 z-30 size-7 rotate-45 border-2 border-white shadow-lg" style="background: {theme.gem};"></div>
	</div>
</div>
