<script lang="ts">
	import Meta from '$lib/components/Meta.svelte';
	import PageHero from '$lib/components/PageHero.svelte';
	import CtaBand from '$lib/components/CtaBand.svelte';
	import { reveal } from '$lib/motion';
	import progress from '$lib/illustrations/progress.svg';
	import TrendUp from 'phosphor-svelte/lib/TrendUp';
	import Lightbulb from 'phosphor-svelte/lib/Lightbulb';
	import Shapes from 'phosphor-svelte/lib/Shapes';
	import ChartBar from 'phosphor-svelte/lib/ChartBar';
	import Flag from 'phosphor-svelte/lib/Flag';
	import SealCheck from 'phosphor-svelte/lib/SealCheck';
	const blocks = [
		{ icon: TrendUp, tone: 'pastel-grape', color: '#5b5ce2', title: 'Soal menyesuaikan anak', body: 'Challenge dimulai dengan soal pemanasan yang sedikit lebih mudah. Tiga jawaban benar berturut-turut membuat soal berikutnya lebih menantang; dua jawaban keliru berturut-turut membuatnya lebih ringan. Skill yang paling perlu dilatih muncul lebih sering.' },
		{ icon: Lightbulb, tone: 'pastel-sun', color: '#c9501c', title: 'Keliru, petunjuk, coba lagi', body: 'Saat jawaban keliru, anak melihat "Hampir benar. Coba lihat lagi petunjuknya." beserta petunjuk, lalu boleh mencoba sekali lagi. Setelah itu penjelasan ditampilkan, jadi setiap soal tetap menjadi kesempatan belajar.' },
		{ icon: Shapes, tone: 'pastel-sky', color: '#3a8ee8', title: 'Jenis soal sesuai usia', body: 'Anak TK menjawab dengan mengetuk gambar, menghitung benda, dan mengurutkan kejadian tanpa mengetik. Anak SD mendapat pilihan ganda, isian angka dengan tombol besar, memasangkan, dan soal cerita. Setiap soal bisa dibacakan.' },
		{ icon: ChartBar, tone: 'pastel-mint', color: '#15804a', title: 'Penguasaan per skill', body: 'Penguasaan dihitung dari 20 jawaban pertama terbaru pada setiap skill, dengan jawaban terbaru lebih berbobot. Jawaban benar setelah petunjuk dihitung setengah. Laporan menunjukkan skill yang kuat (70% ke atas), berkembang (40–70%), dan perlu latihan (di bawah 40%), per pelajaran dan per bab.' },
		{ icon: Flag, tone: 'pastel-peach', color: '#c9501c', title: 'Misi dari orang tua', body: 'Saran di laporan bisa dijadikan misi, misalnya "5 challenge soal cerita minggu ini". Misi muncul paling atas di beranda anak, soalnya fokus pada skill itu, dan anak mendapat lencana saat selesai.' },
		{ icon: SealCheck, tone: 'pastel-rose', color: '#e5487a', title: 'Soal ditinjau sebelum tayang', body: 'Setiap soal melalui tinjauan: kebenaran jawaban dan penjelasan, kesesuaian kurikulum, keterbacaan untuk anak, dan aksesibilitas.' }
	];
</script>

<Meta title="Cara kerja" description="Bagaimana EduPlay menyusun challenge, memberi umpan balik, menghitung penguasaan skill, dan membuat misi latihan." />
<PageHero title="Cara kerja EduPlay" lead="Setiap challenge adalah rangkaian soal pendek. Di balik layar, setiap jawaban menjadi bukti belajar untuk satu skill." image={progress} />

<div class="reveal-stagger mx-auto grid max-w-6xl gap-5 px-4 sm:px-6 md:grid-cols-2" use:reveal>
	{#each blocks as b}
		<section class="{b.tone} lift relative overflow-hidden rounded-[1.6rem] p-6 sm:p-8">
			<span class="pointer-events-none absolute -right-6 -bottom-8 -rotate-12 opacity-[0.1]" aria-hidden="true"><b.icon size={150} weight="fill" color={b.color} /></span>
			<span class="relative grid size-12 place-items-center rounded-full bg-white shadow-sm"><b.icon size={26} weight="duotone" color={b.color} aria-hidden="true" /></span>
			<h2 class="relative mt-4 text-xl">{b.title}</h2>
			<p class="relative mt-2 font-medium text-ink-soft">{b.body}</p>
		</section>
	{/each}
</div>

<CtaBand />
