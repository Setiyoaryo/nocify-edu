<script lang="ts">
	import Meta from '$lib/components/Meta.svelte';
	import HeroDemo from '$lib/components/HeroDemo.svelte';
	import MasteryBar from '$lib/components/MasteryBar.svelte';
	import Figure from '$lib/components/Figure.svelte';
	import { track } from '$lib/api';
	import { faqs } from '$lib/content/faq';
	import { monthlyPriceLabel } from '$lib/pricing';
	import { onMount } from 'svelte';
	import { reveal } from '$lib/motion';
	import heroKids from '$lib/illustrations/hero-kids.svg';
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
	import FunCard from '$lib/components/FunCard.svelte';

	onMount(() => track('landing_visit'));

	const subjects = [
		{ name: 'Matematika', icon: MathOperations, tone: 'grape', color: '#5b5ce2', desc: 'TK B, Kelas 1, Kelas 4', skills: ['Membilang', 'Penjumlahan', 'Perkalian', 'Pecahan', 'Nilai tempat'] },
		{ name: 'Bahasa Indonesia', icon: BookOpenText, tone: 'peach', color: '#c9501c', desc: 'TK B, Kelas 1', skills: ['Huruf awal', 'Suku kata', 'Kosakata', 'Urutan cerita'] },
		{ name: 'IPAS', icon: Plant, tone: 'mint', color: '#15804a', desc: 'Kelas 3', skills: ['Bagian tumbuhan', 'Wujud benda'] }
	];
	const steps = [
		{ icon: UserPlus, title: 'Buat profil anak', body: 'Cukup nama panggilan dan kelas. Tanpa foto, alamat, atau nomor telepon anak.' },
		{ icon: GameController, title: 'Anak main challenge', body: 'Soal bergambar yang bisa dibacakan. Tingkat kesulitannya menyesuaikan jawaban anak.' },
		{ icon: ChartLineUp, title: 'Anda lihat perkembangannya', body: 'Skill yang kuat, yang perlu dilatih, lalu jadikan misi latihan untuk minggu ini.' }
	];
	const values = [
		{ icon: Lightbulb, tone: 'sun', color: '#c9501c', title: 'Salah itu wajar', body: 'Jawaban keliru dibalas petunjuk dan kesempatan mencoba lagi, lalu penjelasan.' },
		{ icon: SpeakerHigh, tone: 'sky', color: '#3a8ee8', title: 'Bisa dibacakan', body: 'Anak TK yang belum lancar membaca cukup mengetuk tombol suara.' },
		{ icon: Timer, tone: 'rose', color: '#e5487a', title: 'Sesi singkat', body: '5–12 soal per challenge sesuai usia, lalu selesai. Lebih lama tidak selalu lebih baik.' },
		{ icon: ShieldCheck, tone: 'mint', color: '#15804a', title: 'Tanpa iklan & peringkat', body: 'Tidak ada leaderboard atau lootbox. Data anak hanya untuk belajar.' }
	];
</script>

<Meta
	title="EduPlay"
	description="Challenge singkat Matematika, Bahasa Indonesia, dan IPAS untuk anak TK–SD. Orang tua melihat skill yang sudah dikuasai dan yang perlu dilatih."
/>

<!-- Hero: brand band with a curved edge -->
<section class="relative overflow-hidden bg-primary text-white">
	<div class="mx-auto grid max-w-6xl items-center gap-10 px-4 pt-12 pb-24 sm:px-6 md:pb-32 lg:grid-cols-[1.05fr_1fr] lg:pt-16">
		<div>
			<p class="eyebrow-chip anim-rise bg-white/15 text-white">Latihan pelajaran sekolah untuk TK & SD</p>
			<h1 class="anim-rise mt-5 text-[2.35rem] sm:text-5xl lg:text-[3.6rem]" style="--d:80ms">Anak bermain.<br />Anda tahu apa yang sudah ia kuasai.</h1>
			<p class="anim-rise mt-5 max-w-xl text-lg text-white/85" style="--d:160ms">
				Challenge 5–10 menit untuk pelajaran sekolah. Setiap jawaban menjadi catatan per skill, jadi Anda tahu apa yang sudah bisa dan apa yang perlu dilatih minggu ini.
			</p>
			<div class="anim-rise mt-8 flex flex-wrap gap-3" style="--d:240ms">
				<a href="/register" class="btn btn-sun text-[1.05rem]">Daftar gratis</a>
				<a href="#coba" class="btn btn-ghost-light text-[1.05rem]">Coba contoh soal</a>
			</div>
			<ul class="anim-rise mt-7 flex flex-wrap gap-x-6 gap-y-2 text-sm font-semibold text-white/90" style="--d:320ms">
				{#each ['Gratis untuk 1 anak', 'Tanpa kartu kredit', 'Tanpa iklan'] as t}
					<li class="flex items-center gap-1.5"><CheckCircle size={18} weight="fill" class="text-sun" aria-hidden="true" />{t}</li>
				{/each}
			</ul>
		</div>
		<div class="anim-rise-scale relative" style="--d:180ms">
			<div class="rounded-[2rem] bg-white p-6 shadow-[0_30px_60px_-20px_rgb(0_0_0/0.35)] sm:p-8">
				<img src={heroKids} alt="" class="mx-auto h-auto w-full max-w-md" width="560" height="360" />
			</div>
			<!-- Floating proof points: what a parent actually sees -->
			<div class="anim-pop absolute -bottom-6 -left-2 sm:-left-6" style="--d:750ms"><div class="card anim-float flex items-center gap-3 px-4 py-3 text-ink">
				<span class="grid size-10 place-items-center rounded-full bg-leaf-tint text-leaf-deep"><TrendUp size={22} weight="bold" aria-hidden="true" /></span>
				<span class="text-sm leading-tight"><b class="block">Penjumlahan 1–20</b><span class="text-ink-soft">naik ke 86%, kuat</span></span>
			</div></div>
			<div class="anim-pop absolute -top-4 -right-2 hidden sm:block" style="--d:950ms"><div class="card anim-float flex items-center gap-2 px-4 py-2.5 text-ink" style="--d:-3s">
				<Star size={20} weight="fill" class="text-sun-deep" aria-hidden="true" /><span class="text-sm font-bold">3 bintang!</span>
			</div></div>
		</div>
	</div>
	<svg class="absolute inset-x-0 bottom-0 block h-12 w-full md:h-20" viewBox="0 0 1440 80" preserveAspectRatio="none" aria-hidden="true">
		<path d="M0 80V36C240 6 480-6 720 6s480 50 720 30v44H0Z" fill="var(--color-canvas)" />
	</svg>
</section>

<!-- Subjects -->
<section class="mx-auto max-w-6xl px-4 pt-14 sm:px-6">
	<div class="flex flex-wrap items-end justify-between gap-4">
		<div>
			<h2 class="text-3xl sm:text-4xl">Pelajaran yang tersedia</h2>
			<p class="mt-2 max-w-xl text-ink-soft">Disusun per kelas, bab, dan skill. Kelas dan materi lain menyusul bertahap.</p>
		</div>
		<a href="/subjects" class="inline-flex items-center gap-1.5 font-bold text-primary-deep hover:underline">Lihat semua skill</a>
	</div>
	<div class="reveal-stagger mt-8 grid gap-5 md:grid-cols-3" use:reveal>
		{#each subjects as s}
			<FunCard size="lg" tone={s.tone as any} icon={s.icon} iconColor={s.color} title={s.name} desc={s.desc} cta="Lihat skill" href="/subjects">
				<ul class="flex flex-wrap gap-1.5">
					{#each s.skills as k}<li class="rounded-full bg-white/80 px-3 py-1 text-sm font-semibold">{k}</li>{/each}
				</ul>
			</FunCard>
		{/each}
	</div>
</section>

<!-- Live demo -->
<section id="coba" class="mx-auto max-w-6xl scroll-mt-20 px-4 pt-20 sm:px-6">
	<div class="max-w-2xl">
		<h2 class="text-3xl sm:text-4xl">Coba satu challenge sekarang</h2>
		<p class="mt-2 text-ink-soft">Jawab seperti anak Anda, lalu lihat laporan di sebelahnya terisi.</p>
	</div>
	<div class="reveal mt-8" use:reveal><HeroDemo /></div>
</section>

<!-- How it works -->
<section class="mx-auto max-w-6xl px-4 pt-20 sm:px-6">
	<h2 class="text-3xl sm:text-4xl">Tiga langkah, sepuluh menit sehari</h2>
	<ol class="reveal-stagger mt-10 grid gap-6 md:grid-cols-3" use:reveal>
		{#each steps as st, i}
			<li class="card lift relative p-6">
				<span class="absolute top-6 right-6 text-5xl font-extrabold text-paper-deep" aria-hidden="true">{i + 1}</span>
				<span class="grid size-12 place-items-center rounded-2xl bg-primary text-white"><st.icon size={26} weight="bold" aria-hidden="true" /></span>
				<h3 class="mt-5 text-xl">{st.title}</h3>
				<p class="mt-2 text-ink-soft">{st.body}</p>
			</li>
		{/each}
	</ol>
</section>

<!-- Report showcase -->
<section class="mx-auto mt-20 max-w-6xl px-4 sm:px-6">
	<div class="reveal card grid items-center gap-10 overflow-hidden p-6 sm:p-10 lg:grid-cols-2" use:reveal>
		<div>
			<span class="eyebrow-chip bg-primary-tint text-primary-deep">Laporan orang tua</span>
			<h2 class="mt-4 text-3xl sm:text-4xl">Bukan sekadar nilai</h2>
			<p class="mt-4 max-w-lg text-ink-soft">"Matematika 80" tidak memberi tahu apa yang harus dilatih. EduPlay mencatat penguasaan per skill, lalu memberi saran yang bisa langsung dijadikan misi untuk anak.</p>
			<div class="mt-6 space-y-4">
				<MasteryBar label="Penjumlahan 1–20" value={88} />
				<MasteryBar label="Perkalian" value={61} />
				<MasteryBar label="Soal cerita" value={34} />
			</div>
			<div class="mt-6 flex items-start gap-3 rounded-2xl bg-sun-tint p-4">
				<Flag size={22} weight="fill" class="mt-0.5 shrink-0 text-tangerine-deep" aria-hidden="true" />
				<p class="text-sm"><b>Misi minggu ini:</b> 5 challenge soal cerita. Anak melihat misi ini di beranda mereka.</p>
			</div>
		</div>
		<img src={parentChild} alt="" class="mx-auto w-full max-w-md" loading="lazy" width="480" height="360" />
	</div>
</section>

<!-- Values -->
<section class="mx-auto max-w-6xl px-4 pt-20 sm:px-6">
	<h2 class="max-w-2xl text-3xl sm:text-4xl">Dibuat untuk anak belajar, bukan untuk menahan mereka di layar</h2>
	<div class="reveal-stagger mt-10 grid gap-5 sm:grid-cols-2 lg:grid-cols-4" use:reveal>
		{#each values as v}
			<FunCard tone={v.tone as any} icon={v.icon} iconColor={v.color} title={v.title} desc={v.body} />
		{/each}
	</div>
</section>

<!-- Pricing teaser -->
<section class="mx-auto max-w-6xl px-4 pt-20 sm:px-6">
	<div class="reveal grid items-center gap-8 rounded-[2rem] bg-night p-6 text-white sm:p-10 lg:grid-cols-[1fr_auto]" use:reveal>
		<div class="flex items-start gap-5">
			<span class="anim-float hidden sm:block"><Figure name="rocket" size={64} /></span>
			<div>
				<h2 class="text-3xl">Mulai gratis, berlangganan saat laporannya terasa berguna</h2>
				<p class="mt-3 text-white/75">Paket bulanan {monthlyPriceLabel}: challenge tanpa batas, laporan per skill lengkap, misi latihan, dan hingga 5 profil anak.</p>
			</div>
		</div>
		<div class="flex flex-wrap gap-3">
			<a href="/register" class="btn btn-sun">Daftar gratis</a>
			<a href="/pricing" class="btn btn-ghost-light">Lihat harga</a>
		</div>
	</div>
</section>

<!-- FAQ on a brand band -->
<section class="mt-20 bg-primary py-16 text-white">
	<div class="mx-auto max-w-3xl px-4 sm:px-6">
		<h2 class="text-center text-3xl sm:text-4xl">Pertanyaan yang sering ditanyakan</h2>
		<div class="reveal-stagger mt-8 space-y-3" use:reveal>
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
