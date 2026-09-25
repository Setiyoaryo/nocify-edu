<script lang="ts">
	import Meta from '$lib/components/Meta.svelte';
	import PageHero from '$lib/components/PageHero.svelte';
	import progress from '$lib/illustrations/progress.svg';
	import TrendUp from 'phosphor-svelte/lib/TrendUp';
	import Lightbulb from 'phosphor-svelte/lib/Lightbulb';
	import Shapes from 'phosphor-svelte/lib/Shapes';
	import ChartBar from 'phosphor-svelte/lib/ChartBar';
	import Flag from 'phosphor-svelte/lib/Flag';
	import SealCheck from 'phosphor-svelte/lib/SealCheck';
	const blocks = [
		{ icon: TrendUp, tint: 'bg-primary-tint text-primary-deep', title: 'Soal menyesuaikan anak', body: 'Challenge dimulai dengan soal pemanasan yang sedikit lebih mudah. Tiga jawaban benar berturut-turut membuat soal berikutnya lebih menantang; dua jawaban keliru berturut-turut membuatnya lebih ringan. Skill yang paling perlu dilatih muncul lebih sering.' },
		{ icon: Lightbulb, tint: 'bg-sun-tint text-ink', title: 'Keliru, petunjuk, coba lagi', body: 'Saat jawaban keliru, anak melihat "Hampir benar. Coba lihat lagi petunjuknya." beserta petunjuk, lalu boleh mencoba sekali lagi. Setelah itu penjelasan ditampilkan, jadi setiap soal tetap menjadi kesempatan belajar.' },
		{ icon: Shapes, tint: 'bg-sky-tint text-sky', title: 'Jenis soal sesuai usia', body: 'Anak TK menjawab dengan mengetuk gambar, menghitung benda, dan mengurutkan kejadian tanpa mengetik. Anak SD mendapat pilihan ganda, isian angka dengan tombol besar, memasangkan, dan soal cerita. Setiap soal bisa dibacakan.' },
		{ icon: ChartBar, tint: 'bg-leaf-tint text-leaf-deep', title: 'Penguasaan per skill', body: 'Penguasaan dihitung dari 20 jawaban pertama terbaru pada setiap skill, dengan jawaban terbaru lebih berbobot. Jawaban benar setelah petunjuk dihitung setengah. Laporan menunjukkan skill yang kuat (70% ke atas), berkembang (40–70%), dan perlu latihan (di bawah 40%), per pelajaran dan per bab.' },
		{ icon: Flag, tint: 'bg-tangerine-tint text-tangerine-deep', title: 'Misi dari orang tua', body: 'Saran di laporan bisa dijadikan misi, misalnya "5 challenge soal cerita minggu ini". Misi muncul paling atas di beranda anak, soalnya fokus pada skill itu, dan anak mendapat lencana saat selesai.' },
		{ icon: SealCheck, tint: 'bg-rose-tint text-rose', title: 'Soal ditinjau sebelum tayang', body: 'Setiap soal melalui tinjauan: kebenaran jawaban dan penjelasan, kesesuaian kurikulum, keterbacaan untuk anak, dan aksesibilitas.' }
	];
</script>

<Meta title="Cara kerja" description="Bagaimana EduPlay menyusun challenge, memberi umpan balik, menghitung penguasaan skill, dan membuat misi latihan." />
<PageHero title="Cara kerja EduPlay" lead="Setiap challenge adalah rangkaian soal pendek. Di balik layar, setiap jawaban menjadi bukti belajar untuk satu skill." image={progress} />

<div class="mx-auto grid max-w-6xl gap-5 px-4 sm:px-6 md:grid-cols-2">
	{#each blocks as b}
		<section class="card p-6 sm:p-8">
			<span class="grid size-12 place-items-center rounded-2xl {b.tint}"><b.icon size={26} weight="duotone" aria-hidden="true" /></span>
			<h2 class="mt-4 text-xl">{b.title}</h2>
			<p class="mt-2 text-ink-soft">{b.body}</p>
		</section>
	{/each}
	<p class="pt-4"><a href="/register" class="btn btn-sun">Daftar gratis</a></p>
</div>
