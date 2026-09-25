<script lang="ts">
	import Meta from '$lib/components/Meta.svelte';
	import PageHero from '$lib/components/PageHero.svelte';
	import CtaBand from '$lib/components/CtaBand.svelte';
	import parentChild from '$lib/illustrations/parent-child.svg';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	import ShieldCheck from 'phosphor-svelte/lib/ShieldCheck';
	import HandHeart from 'phosphor-svelte/lib/HandHeart';
	const sees = ['Aktivitas terakhir, streak belajar, dan jumlah challenge minggu ini', 'Akurasi jawaban pertama dan tren mingguan', 'Penguasaan per pelajaran, per bab, dan per skill', 'Riwayat challenge beserta skornya', 'Saran latihan yang bisa langsung dijadikan misi untuk anak'];
</script>

<Meta title="Untuk orang tua" description="Apa yang Anda lihat sebagai orang tua, bagaimana data anak dijaga, dan cara mendampingi anak belajar dengan EduPlay." />
<PageHero title="Untuk orang tua" lead="Anda pemilik akun. Anak hanya melihat challenge, misi, bintang, dan stikernya. Laporan, pengaturan, dan langganan ada di area orang tua." image={parentChild} />

<div class="mx-auto grid max-w-6xl gap-5 px-4 sm:px-6 lg:grid-cols-3">
	<section class="pastel-grape rounded-[1.6rem] p-6 sm:p-8 lg:col-span-2">
		<h2 class="text-2xl">Yang Anda lihat</h2>
		<ul class="mt-5 space-y-3">
			{#each sees as s}<li class="flex gap-3"><CheckCircle size={22} weight="fill" class="shrink-0 text-primary" aria-hidden="true" />{s}</li>{/each}
		</ul>
	</section>
	<section class="pastel-mint rounded-[1.6rem] p-6 sm:p-8">
		<span class="grid size-12 place-items-center rounded-full bg-white text-leaf-deep shadow-sm"><ShieldCheck size={26} weight="duotone" aria-hidden="true" /></span>
		<h2 class="mt-4 text-xl">Data anak</h2>
		<p class="mt-2 text-ink-soft">Hanya nama panggilan, kelas, avatar, PIN opsional, dan catatan jawaban. Tanpa foto, alamat, nomor telepon, atau lokasi. Tidak dipakai untuk iklan dan bisa Anda hapus kapan saja.</p>
	</section>
	<section class="pastel-sun rounded-[1.6rem] p-6 sm:p-8 lg:col-span-3">
		<div class="flex items-start gap-4">
			<span class="grid size-12 shrink-0 place-items-center rounded-full bg-white text-tangerine-deep shadow-sm"><HandHeart size={26} weight="duotone" aria-hidden="true" /></span>
			<div>
				<h2 class="text-xl">Mendampingi anak</h2>
				<p class="mt-2 max-w-3xl text-ink-soft">Sepuluh menit sehari lebih bermanfaat daripada satu jam seminggu sekali. Saat laporan menunjukkan satu skill perlu latihan, jadikan misi, lalu duduk bersama anak untuk satu challenge dan bahas penjelasannya.</p>
			</div>
		</div>
	</section>
</div>

<CtaBand />
