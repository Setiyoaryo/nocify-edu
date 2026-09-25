<script lang="ts">
	import Meta from '$lib/components/Meta.svelte';
	import HeroDemo from '$lib/components/HeroDemo.svelte';
	import MasteryBar from '$lib/components/MasteryBar.svelte';
	import FunCard from '$lib/components/FunCard.svelte';
	import PhoneMock from '$lib/components/PhoneMock.svelte';
	import { track } from '$lib/api';
	import { faqs } from '$lib/content/faq';
	import { monthlyPriceLabel } from '$lib/pricing';
	import { reveal } from '$lib/motion';
	import { onMount } from 'svelte';
	import parentChild from '$lib/illustrations/parent-child.svg';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	import MathOperations from 'phosphor-svelte/lib/MathOperations';
	import BookOpenText from 'phosphor-svelte/lib/BookOpenText';
	import Plant from 'phosphor-svelte/lib/Plant';
	import UserPlus from 'phosphor-svelte/lib/UserPlus';
	import GameController from 'phosphor-svelte/lib/GameController';
	import ChartLineUp from 'phosphor-svelte/lib/ChartLineUp';
	import Lightbulb from 'phosphor-svelte/lib/Lightbulb';
	import SpeakerHigh from 'phosphor-svelte/lib/SpeakerHigh';
	import ShieldCheck from 'phosphor-svelte/lib/ShieldCheck';
	import Timer from 'phosphor-svelte/lib/Timer';
	import TrendUp from 'phosphor-svelte/lib/TrendUp';
	import Star from 'phosphor-svelte/lib/Star';
	import Plus from 'phosphor-svelte/lib/Plus';
	import Flag from 'phosphor-svelte/lib/Flag';
	import Books from 'phosphor-svelte/lib/Books';
	import Shapes from 'phosphor-svelte/lib/Shapes';
	import Rocket from 'phosphor-svelte/lib/Rocket';
	import CheckFat from 'phosphor-svelte/lib/CheckFat';

	onMount(() => track('landing_visit'));

	const facts = [
		{ icon: Books, tone: 'bg-primary-tint text-primary-deep', value: '3', label: 'pelajaran sekolah' },
		{ icon: Shapes, tone: 'bg-tangerine-tint text-tangerine-deep', value: '7', label: 'jenis soal interaktif' },
		{ icon: Timer, tone: 'bg-leaf-tint text-leaf-deep', value: '5–10', label: 'menit per sesi' },
		{ icon: ShieldCheck, tone: 'bg-rose-tint text-rose', value: '0', label: 'iklan untuk anak' }
	];
	const subjects = [
		{ name: 'Matematika', icon: MathOperations, tone: 'grape', color: '#5b5ce2', desc: 'TK B, Kelas 1, Kelas 4', skills: ['Membilang', 'Penjumlahan', 'Perkalian', 'Pecahan', 'Nilai tempat'] },
		{ name: 'Bahasa Indonesia', icon: BookOpenText, tone: 'peach', color: '#c9501c', desc: 'TK B, Kelas 1', skills: ['Huruf awal', 'Suku kata', 'Kosakata', 'Urutan cerita'] },
		{ name: 'IPAS', icon: Plant, tone: 'mint', color: '#15804a', desc: 'Kelas 3', skills: ['Bagian tumbuhan', 'Wujud benda'] }
	] as const;
	const steps = [
		{ icon: UserPlus, tone: 'pastel-grape', color: '#5b5ce2', title: 'Buat profil anak', body: 'Cukup nama panggilan dan kelas. Tanpa foto atau data pribadi lain.' },
		{ icon: GameController, tone: 'pastel-peach', color: '#c9501c', title: 'Anak main challenge', body: 'Soal bergambar yang bisa dibacakan dan menyesuaikan kemampuan anak.' },
		{ icon: ChartLineUp, tone: 'pastel-mint', color: '#15804a', title: 'Anda lihat perkembangannya', body: 'Skill yang kuat dan yang perlu dilatih, lalu jadikan misi minggu ini.' }
	];
	const values = [
		{ icon: Lightbulb, tone: 'sun', color: '#c9501c', title: 'Salah itu wajar', body: 'Jawaban keliru dibalas petunjuk dan kesempatan mencoba lagi.' },
		{ icon: SpeakerHigh, tone: 'sky', color: '#3a8ee8', title: 'Bisa dibacakan', body: 'Anak TK cukup mengetuk tombol suara.' },
		{ icon: Timer, tone: 'rose', color: '#e5487a', title: 'Sesi singkat', body: '5–12 soal sesuai usia, lalu selesai.' },
		{ icon: ShieldCheck, tone: 'mint', color: '#15804a', title: 'Tanpa iklan', body: 'Tidak ada leaderboard atau lootbox.' }
	] as const;
</script>

<Meta
	title="EduPlay"
	description="Challenge singkat Matematika, Bahasa Indonesia, dan IPAS untuk anak TK–SD. Orang tua melihat skill yang sudah dikuasai dan yang perlu dilatih."
/>

<!-- Hero -->
<section class="relative overflow-hidden bg-primary text-white">
	<div class="mx-auto grid max-w-6xl items-center gap-10 px-4 pt-9 pb-28 sm:px-6 md:pb-36 lg:grid-cols-[1.05fr_1fr] lg:gap-6 lg:pt-14">
		<div class="text-center lg:text-left">
			<p class="eyebrow-chip anim-rise bg-white/15 text-white"><Star size={16} weight="fill" class="text-sun" aria-hidden="true" />Latihan seru untuk TK & SD</p>
			<h1 class="anim-rise mt-5 text-[2.15rem] leading-[1.1] sm:text-5xl lg:text-[3.5rem]" style="--d:80ms">Anak bermain. <span class="text-sun">Anda tahu</span> apa yang sudah ia kuasai.</h1>
			<p class="anim-rise mx-auto mt-4 max-w-xl text-base text-white/85 sm:text-lg lg:mx-0" style="--d:160ms">
				Challenge 5–10 menit untuk pelajaran sekolah. Setiap jawaban jadi catatan per skill untuk Anda.
			</p>
			<div class="anim-rise mt-7 grid gap-3 sm:flex sm:justify-center lg:justify-start" style="--d:240ms">
				<a href="/register" class="btn btn-sun text-[1.05rem]">Daftar gratis</a>
				<a href="#coba" class="btn btn-ghost-light text-[1.05rem]">Coba contoh soal</a>
			</div>
			<ul class="anim-rise mt-6 flex flex-wrap justify-center gap-x-5 gap-y-2 text-sm font-semibold text-white/90 lg:justify-start" style="--d:320ms">
				{#each ['Gratis untuk 1 anak', 'Tanpa kartu kredit', 'Tanpa iklan'] as t}
					<li class="flex items-center gap-1.5"><CheckCircle size={18} weight="fill" class="text-sun" aria-hidden="true" />{t}</li>
				{/each}
			</ul>
		</div>

		<div class="anim-rise-scale relative mx-auto w-full max-w-md" style="--d:180ms">
			<!-- soft rings focus the eye on the phone -->
			<div class="pointer-events-none absolute top-1/2 left-1/2 size-[340px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-white/8 sm:size-[420px]" aria-hidden="true"></div>
			<div class="pointer-events-none absolute top-1/2 left-1/2 size-[250px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-white/10 sm:size-[310px]" aria-hidden="true"></div>
			<div class="anim-float relative" style="--d:-2s"><PhoneMock /></div>
			<div class="anim-pop absolute bottom-16 -left-1 sm:bottom-24 sm:-left-6" style="--d:800ms">
				<div class="card anim-float flex items-center gap-2.5 px-3.5 py-2.5 text-ink sm:gap-3 sm:px-4 sm:py-3">
					<span class="grid size-9 place-items-center rounded-full bg-leaf-tint text-leaf-deep sm:size-10"><TrendUp size={20} weight="bold" aria-hidden="true" /></span>
					<span class="text-xs leading-tight sm:text-sm"><b class="block">Penjumlahan</b><span class="text-ink-soft">naik ke 86%, kuat</span></span>
				</div>
			</div>
			<div class="anim-pop absolute top-10 -right-1 sm:top-16 sm:-right-4" style="--d:1000ms">
				<div class="card anim-float flex items-center gap-2 px-3.5 py-2.5 text-ink" style="--d:-3s">
					<span class="grid size-8 place-items-center rounded-full bg-leaf text-white"><CheckFat size={16} weight="fill" aria-hidden="true" /></span>
					<span class="text-xs font-extrabold sm:text-sm">Hebat! Benar.</span>
				</div>
			</div>
		</div>
	</div>
	<svg class="absolute inset-x-0 bottom-0 block h-12 w-full md:h-20" viewBox="0 0 1440 80" preserveAspectRatio="none" aria-hidden="true">
		<path d="M0 80V36C240 6 480-6 720 6s480 50 720 30v44H0Z" fill="var(--color-canvas)" />
	</svg>
</section>

<!-- Facts: overlaps the hero edge -->
<section class="relative z-10 mx-auto -mt-16 max-w-5xl px-4 sm:px-6 md:-mt-20">
	<ul class="reveal-stagger card grid grid-cols-2 gap-px overflow-hidden bg-line p-0 md:grid-cols-4" use:reveal>
		{#each facts as f}
			<li class="flex items-center gap-3 bg-paper p-4 sm:p-5">
				<span class="grid size-11 shrink-0 place-items-center rounded-2xl {f.tone}"><f.icon size={24} weight="duotone" aria-hidden="true" /></span>
				<span class="leading-tight"><b class="block text-2xl font-extrabold">{f.value}</b><span class="text-xs font-semibold text-ink-soft sm:text-sm">{f.label}</span></span>
			</li>
		{/each}
	</ul>
</section>

<!-- Subjects: swipe on phones, grid on larger screens -->
<section class="mx-auto max-w-6xl pt-14 sm:pt-16">
	<div class="flex flex-wrap items-end justify-between gap-3 px-4 sm:px-6">
		<div>
			<h2 class="text-[1.7rem] sm:text-4xl">Pelajaran yang tersedia</h2>
			<p class="mt-1.5 max-w-xl text-ink-soft">Disusun per kelas, bab, dan skill. Materi lain menyusul bertahap.</p>
		</div>
		<a href="/subjects" class="font-bold text-primary-deep hover:underline">Lihat semua skill</a>
	</div>
	<div class="reveal-stagger mt-6 flex snap-x snap-mandatory gap-4 overflow-x-auto px-4 pb-2 [scrollbar-width:none] sm:px-6 md:grid md:grid-cols-3 md:overflow-visible md:pb-0" use:reveal>
		{#each subjects as s}
			<div class="w-[80%] shrink-0 snap-center sm:w-[46%] md:w-auto">
				<FunCard size="lg" tone={s.tone} icon={s.icon} iconColor={s.color} title={s.name} desc={s.desc} cta="Lihat skill" href="/subjects">
					<ul class="flex flex-wrap gap-1.5">
						{#each s.skills as k}<li class="rounded-full bg-white/80 px-3 py-1 text-sm font-semibold">{k}</li>{/each}
					</ul>
				</FunCard>
			</div>
		{/each}
	</div>
	<p class="mt-2 px-4 text-center text-xs font-semibold text-ink-soft md:hidden" aria-hidden="true">Geser untuk melihat pelajaran lain</p>
</section>

<!-- Live demo -->
<section id="coba" class="mx-auto max-w-6xl scroll-mt-20 px-4 pt-16 sm:px-6">
	<div class="max-w-2xl">
		<span class="eyebrow-chip bg-sun-tint text-ink"><GameController size={16} weight="fill" class="text-tangerine" aria-hidden="true" />Coba sekarang, tanpa daftar</span>
		<h2 class="mt-3 text-[1.7rem] sm:text-4xl">Rasakan jadi anak Anda</h2>
		<p class="mt-1.5 text-ink-soft">Jawab dua soal, lalu lihat laporan untuk orang tua langsung terisi.</p>
	</div>
	<div class="reveal mt-6" use:reveal><HeroDemo /></div>
</section>

<!-- How it works -->
<section class="mx-auto max-w-6xl px-4 pt-16 sm:px-6">
	<h2 class="text-[1.7rem] sm:text-4xl">Tiga langkah, sepuluh menit sehari</h2>
	<ol class="reveal-stagger mt-6 grid gap-3 md:grid-cols-3 md:gap-5" use:reveal>
		{#each steps as st, i}
			<li class="{st.tone} lift relative flex items-start gap-4 rounded-[1.4rem] p-5 md:block md:p-6">
				<span class="relative grid size-12 shrink-0 place-items-center rounded-full bg-white shadow-sm">
					<st.icon size={26} weight="duotone" color={st.color} aria-hidden="true" />
					<span class="absolute -top-1 -right-1 grid size-5 place-items-center rounded-full bg-ink text-[11px] font-extrabold text-white">{i + 1}</span>
				</span>
				<div class="md:mt-4">
					<h3 class="text-lg">{st.title}</h3>
					<p class="mt-1 text-sm font-semibold text-ink-soft">{st.body}</p>
				</div>
			</li>
		{/each}
	</ol>
</section>

<!-- Report showcase -->
<section class="mx-auto mt-16 max-w-6xl px-4 sm:px-6">
	<div class="reveal pastel-sky grid items-center gap-8 overflow-hidden rounded-[1.8rem] p-5 sm:p-10 lg:grid-cols-2" use:reveal>
		<div>
			<span class="eyebrow-chip bg-white text-primary-deep">Laporan orang tua</span>
			<h2 class="mt-3 text-[1.7rem] sm:text-4xl">Bukan sekadar nilai</h2>
			<p class="mt-2 max-w-lg text-ink-soft">"Matematika 80" tidak memberi tahu apa yang harus dilatih. EduPlay mencatat penguasaan per skill dan memberi saran yang bisa langsung dijadikan misi.</p>
		</div>
		<div class="card space-y-4 p-5 sm:p-6 lg:row-span-2">
			<p class="flex items-center justify-between text-sm font-bold"><span>Laporan Adi, Kelas 4</span><span class="pill bg-leaf-tint text-leaf-deep">Minggu ini</span></p>
			<MasteryBar label="Penjumlahan 1–20" value={88} />
			<MasteryBar label="Perkalian" value={61} />
			<MasteryBar label="Soal cerita" value={34} />
			<div class="flex items-start gap-3 rounded-2xl bg-sun-tint p-4">
				<Flag size={22} weight="fill" class="mt-0.5 shrink-0 text-tangerine-deep" aria-hidden="true" />
				<p class="text-sm"><b>Misi minggu ini:</b> 5 challenge soal cerita. Anak melihatnya di beranda mereka.</p>
			</div>
		</div>
		<img src={parentChild} alt="" class="mx-auto hidden w-full max-w-xs lg:block" loading="lazy" width="320" height="240" />
	</div>
</section>

<!-- Values -->
<section class="mx-auto max-w-6xl px-4 pt-16 sm:px-6">
	<h2 class="max-w-2xl text-[1.7rem] sm:text-4xl">Dibuat untuk anak belajar, bukan untuk menahan mereka di layar</h2>
	<div class="reveal-stagger mt-6 grid grid-cols-2 gap-3 sm:gap-5 lg:grid-cols-4" use:reveal>
		{#each values as v}
			<FunCard tone={v.tone} icon={v.icon} iconColor={v.color} title={v.title} desc={v.body} />
		{/each}
	</div>
</section>

<!-- Pricing teaser -->
<section class="mx-auto max-w-6xl px-4 pt-16 sm:px-6">
	<div class="reveal relative overflow-hidden rounded-[1.8rem] bg-night p-6 text-white sm:p-10" use:reveal>
		<span class="anim-float pointer-events-none absolute -right-6 -bottom-8 opacity-15" aria-hidden="true"><Rocket size={200} weight="fill" /></span>
		<div class="relative grid items-center gap-6 lg:grid-cols-[1fr_auto]">
			<div>
				<h2 class="text-[1.6rem] sm:text-3xl">Mulai gratis, berlangganan saat laporannya terasa berguna</h2>
				<p class="mt-2 text-white/75">Paket bulanan {monthlyPriceLabel}: challenge tanpa batas, laporan per skill lengkap, misi latihan, dan hingga 5 profil anak.</p>
			</div>
			<div class="grid gap-3 sm:flex">
				<a href="/register" class="btn btn-sun">Daftar gratis</a>
				<a href="/pricing" class="btn btn-ghost-light">Lihat harga</a>
			</div>
		</div>
	</div>
</section>

<!-- FAQ -->
<section class="mt-16 bg-primary py-14 text-white">
	<div class="mx-auto max-w-3xl px-4 sm:px-6">
		<h2 class="text-center text-[1.7rem] sm:text-4xl">Pertanyaan yang sering ditanyakan</h2>
		<div class="reveal-stagger mt-7 space-y-3" use:reveal>
			{#each faqs.slice(0, 5) as [q, a]}
				<details class="group rounded-2xl bg-white text-ink">
					<summary class="flex min-h-14 cursor-pointer list-none items-center justify-between gap-4 px-5 py-3 font-bold">
						{q}<Plus size={20} weight="bold" class="shrink-0 transition-transform group-open:rotate-45" aria-hidden="true" />
					</summary>
					<p class="px-5 pb-5 text-ink-soft">{a}</p>
				</details>
			{/each}
		</div>
		<p class="mt-6 text-center"><a href="/faq" class="font-bold text-sun hover:underline">Lihat semua pertanyaan</a></p>
	</div>
</section>
