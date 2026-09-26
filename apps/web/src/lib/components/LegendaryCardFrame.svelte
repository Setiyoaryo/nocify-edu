<script lang="ts">
	import type { ChildRecap } from '$lib/types';
	import LegendaryHeroScene from './LegendaryHeroScene.svelte';
	import Star from 'phosphor-svelte/lib/Star';
	import Fire from 'phosphor-svelte/lib/Fire';
	import Target from 'phosphor-svelte/lib/Target';
	import BookOpen from 'phosphor-svelte/lib/BookOpen';
	import Sparkle from 'phosphor-svelte/lib/Sparkle';
	import CalendarBlank from 'phosphor-svelte/lib/CalendarBlank';
	import GraduationCap from 'phosphor-svelte/lib/GraduationCap';
	import GameController from 'phosphor-svelte/lib/GameController';
	import ChartBar from 'phosphor-svelte/lib/ChartBar';
	import UsersThree from 'phosphor-svelte/lib/UsersThree';
	import Trophy from 'phosphor-svelte/lib/Trophy';
	import PenNib from 'phosphor-svelte/lib/PenNib';

	let { recap }: { recap: ChildRecap } = $props();

	const theme = $derived.by(() => {
		switch (recap.archetype.code) {
			case 'ksatria_api': return { accent: '#ff7a45', soft: '#fff1e9', deep: '#8f2d16', banner: 'from-[#7c2d12] via-[#ea580c] to-[#f59e0b]' };
			case 'detektif_elang': return { accent: '#38bdf8', soft: '#e8f7ff', deep: '#155e75', banner: 'from-[#0c4a6e] via-[#0284c7] to-[#38bdf8]' };
			case 'penyihir_angka': return { accent: '#a855f7', soft: '#f4e8ff', deep: '#6b21a8', banner: 'from-[#581c87] via-[#9333ea] to-[#c084fc]' };
			case 'penjelajah_kata': return { accent: '#34d399', soft: '#e7fff5', deep: '#065f46', banner: 'from-[#065f46] via-[#059669] to-[#34d399]' };
			default: return { accent: '#f59e0b', soft: '#fff7dc', deep: '#92400e', banner: 'from-[#7c2d12] via-[#f59e0b] to-[#fcd34d]' };
		}
	});
</script>

<article
	class="relative w-full overflow-hidden rounded-[2.25rem] border border-white/80 bg-[#fffaf0] shadow-[0_24px_60px_rgba(27,29,58,0.20)]"
	style="--accent:{theme.accent}; --soft:{theme.soft}; --deep:{theme.deep};"
>
	<div class="pointer-events-none absolute inset-0 opacity-70" style="background:radial-gradient(circle at 10% 0%, var(--soft), transparent 35%),radial-gradient(circle at 95% 20%, #fff4bf, transparent 30%);"></div>
	<div class="pointer-events-none absolute -right-14 top-24 size-44 rounded-full border-[18px] border-white/35"></div>
	<div class="pointer-events-none absolute -left-20 bottom-28 size-56 rounded-full border-[22px] border-white/25"></div>

	<div class="relative flex items-center justify-between px-4 pt-4 sm:px-6">
		<div class="flex items-center gap-2">
			<div class="grid size-10 place-items-center rounded-2xl bg-[#1b1d3a] text-[#ffc857] shadow-sm">
				<Sparkle size={20} weight="fill" />
			</div>
			<div>
				<div class="font-display text-base font-black text-[#3f40c2]">EduPlay</div>
				<div class="text-[0.62rem] font-bold text-[#6b7087]">Belajar jadi lebih seru</div>
			</div>
		</div>
		<div class="inline-flex items-center gap-1.5 rounded-full border border-black/5 bg-white/90 px-3 py-1.5 text-[0.65rem] font-black text-[#4b5068] shadow-sm">
			<CalendarBlank size={13} weight="bold" />
			{recap.archetype.date_formatted}
		</div>
	</div>

	<div class="relative px-4 pt-4 sm:px-6">
		<div class="rounded-[1.75rem] bg-gradient-to-br from-white/85 to-white/45 p-2 ring-1 ring-black/5">
			<LegendaryHeroScene child={recap.child} archetype={recap.archetype} />
		</div>
	</div>

	<div class="relative z-10 -mt-7 flex flex-col items-center px-4 sm:-mt-9">
		<div class="flex items-center gap-[-4px]">
			<span class="h-5 w-20 origin-right -rotate-6 rounded-full bg-gradient-to-r from-transparent via-[#f6c453] to-[#c47a10] shadow-sm"></span>
			<div class="relative rounded-[1.35rem] bg-gradient-to-br {theme.banner} px-7 py-3 shadow-[0_10px_24px_rgba(74,44,8,0.28)] ring-4 ring-[#ffe39a]">
				<div class="absolute -top-3 left-1/2 -translate-x-1/2 grid size-9 place-items-center rounded-full bg-gradient-to-br from-[#fff6a7] to-[#f3a400] text-[#8b5d00] shadow-md ring-2 ring-white">
					<Star size={18} weight="fill" />
				</div>
				<div class="pt-1 text-center font-display text-xl font-black tracking-[0.08em] text-white drop-shadow sm:text-2xl">
					{recap.archetype.banner}
				</div>
				<div class="absolute -bottom-3 left-1/2 -translate-x-1/2 rounded-full bg-gradient-to-r from-[#fff1a6] via-[#ffd45c] to-[#f0a71e] px-5 py-1 text-[0.62rem] font-black uppercase tracking-[0.14em] text-[#5b4213] shadow-md ring-1 ring-[#c98914]">
					{recap.archetype.sub_banner}
				</div>
			</div>
			<span class="h-5 w-20 origin-left rotate-6 rounded-full bg-gradient-to-l from-transparent via-[#f6c453] to-[#c47a10] shadow-sm"></span>
		</div>
	</div>

	<div class="relative z-10 mx-4 mt-7 rounded-[1.5rem] border border-black/5 bg-white/90 p-3.5 shadow-sm sm:mx-6">
		<div class="grid grid-cols-[auto_1fr] gap-3">
			<div class="grid size-14 place-items-center rounded-2xl bg-[#1b1d3a] text-white ring-4 ring-[#ffe39a]">
				<span class="text-[0.56rem] font-black uppercase tracking-wider text-[#ffc857]">Level</span>
				<span class="font-display text-xl font-black leading-none">{recap.child.level}</span>
			</div>
			<div class="min-w-0">
				<div class="truncate font-display text-xl font-black text-[#1b1d3a]">{recap.child.nickname}</div>
				<div class="mt-0.5 flex items-center gap-1.5 text-xs font-bold text-[#6b7087]">
					<PenNib size={12} class="text-[var(--deep)]" />
					{recap.archetype.title}
				</div>
				<p class="mt-2 text-xs font-semibold leading-relaxed text-[#53586e] sm:text-sm">
					“{recap.archetype.quote}”
				</p>
			</div>
		</div>
	</div>

	<div class="relative z-10 mx-4 mt-3 grid grid-cols-2 gap-2 sm:mx-6 sm:grid-cols-4 sm:gap-3">
		{#each [
			{ icon: Star, label: 'Bintang', value: recap.stats.stars, tone: '#fff4cf' },
			{ icon: Fire, label: 'Streak', value: recap.stats.streak, tone: '#ffe8df' },
			{ icon: Target, label: 'Akurasi', value: `${Math.round(recap.stats.accuracy)}%`, tone: '#e8f7ff' },
			{ icon: BookOpen, label: 'Soal', value: recap.stats.total_questions, tone: '#efeaff' }
		] as stat}
			{@const Icon = stat.icon}
			<div class="rounded-[1.25rem] border border-black/5 bg-white/90 p-3 shadow-sm">
				<div class="grid size-9 place-items-center rounded-2xl" style="background:{stat.tone}; color:var(--deep);">
					<Icon size={18} weight="fill" />
				</div>
				<div class="mt-2 font-display text-xl font-black text-[#1b1d3a]">{stat.value}</div>
				<div class="text-[0.68rem] font-extrabold uppercase tracking-wider text-[#6b7087]">{stat.label}</div>
			</div>
		{/each}
	</div>

	<div class="relative z-10 mx-4 mt-3 grid gap-3 sm:mx-6 sm:grid-cols-[1.15fr_0.85fr]">
		<div class="rounded-[1.5rem] border border-black/5 bg-white/90 p-3.5 shadow-sm">
			<div class="flex items-center justify-between gap-3">
				<div class="min-w-0">
					<div class="text-[0.68rem] font-black uppercase tracking-wider text-[#7a7f95]">Menuju tier berikutnya</div>
					<div class="mt-1 truncate font-display text-lg font-black text-[#1b1d3a]">{recap.archetype.next_tier_name}</div>
				</div>
				<div class="rounded-xl bg-[#f0e9ff] px-2.5 py-1 text-xs font-black text-[#6d28d9]">{recap.archetype.progress_percent}%</div>
			</div>
			<div class="mt-2 h-2.5 overflow-hidden rounded-full bg-[#eeeaf6]">
				<div class="h-full rounded-full bg-gradient-to-r from-[#7c3aed] to-[#f59e0b]" style="width:{Math.max(3, Math.min(100, recap.archetype.progress_percent))}%"></div>
			</div>
		</div>
		<div class="rounded-[1.5rem] border border-black/5 bg-[#fff8dd] p-3.5 shadow-sm">
			<div class="flex items-center gap-2">
				<div class="grid size-9 place-items-center rounded-xl bg-white text-[#d99000] shadow-sm">
					<Trophy size={18} weight="fill" />
				</div>
				<div class="text-xs font-black text-[#955f00]">Luar biasa!</div>
			</div>
			<p class="mt-2 text-xs font-bold leading-relaxed text-[#675f4c]">
				Kamu sudah lebih baik dari {recap.archetype.peer_percentile}% teman-teman di EduPlay!
			</p>
		</div>
	</div>

	<div class="relative z-10 mx-4 my-3 flex items-center justify-around gap-2 rounded-[1.35rem] bg-[#1b1d3a] px-2 py-3 text-white sm:mx-6">
		<div class="flex items-center gap-1.5 text-[0.6rem] font-extrabold sm:text-xs"><GraduationCap size={15} class="text-[#ffc857]" /><span>Belajar mudah</span></div>
		<div class="hidden text-white/20 sm:block">•</div>
		<div class="flex items-center gap-1.5 text-[0.6rem] font-extrabold sm:text-xs"><GameController size={15} class="text-[#ffc857]" /><span>Tantangan seru</span></div>
		<div class="hidden text-white/20 sm:block">•</div>
		<div class="flex items-center gap-1.5 text-[0.6rem] font-extrabold sm:text-xs"><ChartBar size={15} class="text-[#ffc857]" /><span>Perkembangan</span></div>
		<div class="hidden text-white/20 sm:block">•</div>
		<div class="hidden items-center gap-1.5 text-[0.6rem] font-extrabold sm:flex sm:text-xs"><UsersThree size={15} class="text-[#ffc857]" /><span>Bersama teman</span></div>
	</div>
</article>

<style>
	article { isolation: isolate; }
</style>
