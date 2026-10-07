<script lang="ts">
	import type { ChildRecap } from '$lib/types';
	import LegendaryHeroScene from './LegendaryHeroScene.svelte';
	import Star from 'phosphor-svelte/lib/Star';
	import Fire from 'phosphor-svelte/lib/Fire';
	import Target from 'phosphor-svelte/lib/Target';
	import BookOpen from 'phosphor-svelte/lib/BookOpen';
	import Sparkle from 'phosphor-svelte/lib/Sparkle';
	import CalendarBlank from 'phosphor-svelte/lib/CalendarBlank';
	import Trophy from 'phosphor-svelte/lib/Trophy';
	import PenNib from 'phosphor-svelte/lib/PenNib';

	let { recap }: { recap: ChildRecap } = $props();

	const theme = $derived.by(() => {
		switch (recap.archetype.code) {
			case 'ksatria_api':
				return { deep: '#9a3412', accent: '#fb923c', soft: '#fff0e6', banner: 'from-[#7c2d12] via-[#ea580c] to-[#f59e0b]' };
			case 'detektif_elang':
				return { deep: '#075985', accent: '#38bdf8', soft: '#e8f7ff', banner: 'from-[#123a7a] via-[#2563eb] to-[#38bdf8]' };
			case 'penyihir_angka':
				return { deep: '#6b21a8', accent: '#c084fc', soft: '#f4e8ff', banner: 'from-[#4c1d95] via-[#7c3aed] to-[#c084fc]' };
			case 'penjelajah_kata':
				return { deep: '#065f46', accent: '#34d399', soft: '#e7fff5', banner: 'from-[#064e3b] via-[#059669] to-[#34d399]' };
			default:
				return { deep: '#92400e', accent: '#f59e0b', soft: '#fff7dc', banner: 'from-[#7c2d12] via-[#f59e0b] to-[#fcd34d]' };
		}
	});

	const stats = $derived([
		{ icon: Star, label: 'Bintang', value: recap.stats.stars, tone: '#fff4cf' },
		{ icon: Fire, label: 'Streak', value: recap.stats.streak, tone: '#ffe8df' },
		{ icon: Target, label: 'Akurasi', value: `${Math.round(recap.stats.accuracy)}%`, tone: '#e8f7ff' },
		{ icon: BookOpen, label: 'Soal', value: recap.stats.total_questions, tone: '#efeaff' }
	]);
</script>

<article
	class="relative mx-auto w-full max-w-[430px] overflow-hidden rounded-[2rem] border-[3px] border-[#e7b84b] bg-[#fffaf0] shadow-[0_22px_55px_rgba(27,29,58,0.18)] sm:rounded-[2.5rem]"
	style="--deep:{theme.deep}; --accent:{theme.accent};"
>
	<div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_10%_0%,rgba(255,255,255,.8),transparent_35%),radial-gradient(circle_at_95%_45%,rgba(255,220,120,.22),transparent_34%)]"></div>

	<!-- 1. Dedicated Top Header Row (NO OVERLAPPING) -->
	<div class="relative z-20 flex items-center justify-between px-3 pt-2.5 pb-1.5 border-b border-black/5">
		<div class="flex items-center gap-1.5">
			<span class="flex size-7 items-center justify-center rounded-xl bg-amber-400 text-black font-black text-xs shadow-xs">👑</span>
			<div>
				<span class="font-display text-xs font-black tracking-wider text-primary-deep uppercase">EDUPLAY</span>
				<span class="block text-[0.56rem] font-bold text-ink-soft leading-none">Belajar Jadi Lebih Seru</span>
			</div>
		</div>
		<div class="flex items-center gap-1 rounded-full border border-amber-300 bg-white px-2.5 py-0.5 text-[0.64rem] font-black text-ink shadow-2xs">
			<CalendarBlank size={12} weight="bold" class="text-amber-600" />
			<span>{recap.archetype.date_formatted}</span>
		</div>
	</div>

	<!-- Motto Sub-Bar -->
	<p class="relative z-20 px-3 py-0.5 text-[0.6rem] font-extrabold italic text-ink-soft text-center">
		"Terus Belajar Terus Berkembang Jadi Versi Terbaikmu!"
	</p>

	<!-- 2. Hero artwork stage -->
	<div class="relative overflow-hidden rounded-[1.75rem] border-2 border-[#efc65f] bg-[#0f4c81] shadow-inner mt-0.5">
		<LegendaryHeroScene child={recap.child} archetype={recap.archetype} />
	</div>
	<!-- Dynamic collectible rarity banner -->
	<div class="relative z-10 -mt-8 px-4 sm:-mt-10 sm:px-6">
		<div class="flex items-center justify-center gap-0">
			<span class="h-7 w-7 min-[380px]:w-14 -rotate-6 rounded-l-full bg-gradient-to-r from-transparent via-[#e6b547] to-[#ad7410] shadow-md sm:w-20"></span>
			<div class="relative min-w-0 rounded-[1.45rem] bg-gradient-to-r {theme.banner} px-5 py-3 shadow-[0_12px_26px_rgba(74,44,8,0.28)] ring-4 ring-[#ffe7a0] sm:px-9">
				<div class="absolute -top-4 left-1/2 grid size-10 -translate-x-1/2 place-items-center rounded-full bg-gradient-to-br from-[#fff8ad] to-[#efa914] text-[#7e5900] shadow-lg ring-2 ring-white">
					<Star size={18} weight="fill" />
				</div>
				<div class="pt-1 text-center font-display text-[clamp(0.95rem,4.5vw,1.55rem)] font-black tracking-[0.08em] text-white drop-shadow-[0_2px_4px_rgba(0,0,0,.4)]">
					{recap.archetype.banner}
				</div>
				<div class="absolute -bottom-4 left-1/2 -translate-x-1/2 rounded-full border border-[#bf8614] bg-gradient-to-r from-[#fff2ab] via-[#ffd45a] to-[#f0a71e] px-3.5 py-1 text-[0.58rem] font-black uppercase tracking-[0.14em] text-[#5b4213] shadow-md whitespace-nowrap sm:px-4 sm:text-[0.64rem]">
					{recap.archetype.sub_banner}
				</div>
			</div>
			<span class="h-7 w-7 min-[380px]:w-14 rotate-6 rounded-r-full bg-gradient-to-l from-transparent via-[#e6b547] to-[#ad7410] shadow-md sm:w-20"></span>
		</div>
	</div>

	<div class="relative z-10 px-2.5 pt-8 sm:px-6 sm:pt-10">
		<section class="rounded-[1.45rem] border border-[#ecd696] bg-gradient-to-b from-[#fffdf7] to-[#fff7df] p-3 shadow-xs">
			<div class="flex items-center gap-3">
				<div class="grid size-14 shrink-0 place-items-center rounded-xl bg-gradient-to-b from-[#7a3f0c] to-[#b96c13] text-white shadow-sm ring-2 ring-[#ffd86b]">
					<span class="text-[0.52rem] font-black uppercase text-[#ffe9a6]">Level</span>
					<span class="font-display text-xl font-black leading-none">{recap.child.level}</span>
				</div>
				<div class="min-w-0 flex-1">
					<h2 class="truncate font-display text-lg font-black text-[#1b1d3a]">{recap.child.nickname}</h2>
					<div class="flex items-center gap-1 text-[0.68rem] font-extrabold text-[var(--deep)] leading-tight">
						<PenNib size={12} class="shrink-0" />
						<span>{recap.archetype.title}</span>
					</div>
				</div>
			</div>
			<p class="mt-2 text-[0.68rem] font-semibold italic leading-relaxed text-[#6a6f81] border-t border-[#ecd696]/60 pt-1.5">
				“{recap.archetype.quote}”
			</p>
		</section>

		<section class="mt-2.5 grid grid-cols-4 gap-1 text-center">
			{#each stats as stat}
				{@const Icon = stat.icon}
				<div class="rounded-xl border border-black/5 bg-white px-1 py-2 shadow-2xs flex flex-col items-center justify-between">
					<div class="grid size-6 place-items-center rounded-lg" style="background:{stat.tone}; color:var(--deep);">
						<Icon size={13} weight="fill" />
					</div>
					<div class="my-0.5 font-display text-sm font-black text-[#1b1d3a] leading-none">{stat.value}</div>
					<div class="text-[0.62rem] font-bold text-[#62677b] leading-tight">{stat.label}</div>
				</div>
			{/each}
		</section>

		<section class="mt-3 grid gap-2.5 sm:grid-cols-[1.1fr_0.9fr] sm:gap-3">
			<div class="rounded-[1.45rem] border border-black/5 bg-white p-3.5 shadow-sm sm:p-4">
				<div class="flex items-center justify-between gap-3">
					<div class="min-w-0">
						<div class="text-[0.65rem] font-black uppercase tracking-wider text-[#7d8295]">Menuju tier berikutnya</div>
						<div class="mt-1 truncate font-display text-lg font-black text-[#1b1d3a]">{recap.archetype.next_tier_name}</div>
					</div>
					<div class="rounded-xl bg-[#f0e9ff] px-2.5 py-1 text-xs font-black text-[#6d28d9]">{recap.archetype.progress_percent}%</div>
				</div>
				<div class="mt-2.5 h-3 overflow-hidden rounded-full bg-[#eeeaf6]">
					<div class="h-full rounded-full bg-gradient-to-r from-[#6d28d9] via-[#a855f7] to-[#f59e0b]" style="width:{Math.max(3, Math.min(100, recap.archetype.progress_percent))}%"></div>
				</div>
			</div>

			<div class="rounded-[1.45rem] border border-[#eedb9d] bg-gradient-to-r from-[#fff7d6] to-[#fffef5] p-3.5 shadow-sm sm:p-4">
				<div class="flex items-center gap-2.5">
					<div class="grid size-10 shrink-0 place-items-center rounded-2xl bg-white text-[#c98700] shadow-sm ring-1 ring-[#f2d98b]">
						<Trophy size={20} weight="fill" />
					</div>
					<div class="text-sm font-black text-[#935b00]">Luar biasa!</div>
				</div>
				<p class="mt-2 text-xs font-bold leading-relaxed text-[#635c49]">
					Kamu sudah lebih baik dari {recap.archetype.peer_percentile}% teman-teman di EduPlay!
				</p>
			</div>
		</section>

		<div class="pb-3 pt-3 sm:pb-4">
			<div class="flex items-center justify-center gap-2 text-[0.66rem] font-extrabold text-[#777b8f]">
				<Sparkle size={13} weight="fill" class="text-[#e5ae28]" />
				<span>Terus belajar, terus berkembang.</span>
				<Sparkle size={13} weight="fill" class="text-[#e5ae28]" />
			</div>
		</div>
	</div>
</article>
