<script lang="ts">
	import Meta from '$lib/components/Meta.svelte';
	import PageHero from '$lib/components/PageHero.svelte';
	import CtaBand from '$lib/components/CtaBand.svelte';
	import FileText from 'phosphor-svelte/lib/FileText';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	import CreditCard from 'phosphor-svelte/lib/CreditCard';
	import WarningCircle from 'phosphor-svelte/lib/WarningCircle';
</script>

<Meta
	title="Syarat dan Ketentuan Layanan"
	description="Syarat dan ketentuan penggunaan platform edukasi EduPlay untuk orang tua dan anak."
/>

<PageHero
	title="Syarat & Ketentuan Layanan"
	lead="Panduan dan ketentuan bagi orang tua dan wali dalam menggunakan platform belajar interaktif EduPlay."
/>

<div class="mx-auto max-w-4xl space-y-8 px-4 sm:px-6">
	<article class="card space-y-4 p-6 sm:p-8">
		<h2 class="text-xl font-bold flex items-center gap-2">
			<FileText size={24} class="text-primary-deep" aria-hidden="true" />
			1. Penerimaan Ketentuan
		</h2>
		<p class="text-ink-soft leading-relaxed text-sm">
			Dengan mendaftar, mengakses, atau menggunakan layanan EduPlay, Anda menyatakan bahwa Anda adalah orang tua atau wali yang sah dari anak yang didaftarkan, telah membaca, memahami, dan menyetujui seluruh ketentuan ini beserta Kebijakan Privasi kami.
		</p>
	</article>

	<article class="card space-y-4 p-6 sm:p-8">
		<h2 class="text-xl font-bold flex items-center gap-2">
			<CheckCircle size={24} class="text-primary-deep" aria-hidden="true" />
			2. Akun dan Tanggung Jawab Orang Tua
		</h2>
		<div class="space-y-3 text-ink-soft leading-relaxed text-sm">
			<p>
				Orang tua atau wali bertindak sebagai pemegang akun resmi. Anda bertanggung jawab menjaga kerahasiaan kata sandi akun orang tua dan mengawasi penggunaan platform oleh anak.
			</p>
			<p>
				Setiap profil anak dapat dilengkapi dengan PIN 4-angka untuk memastikan anak tidak sengaja membuka profil saudaranya atau pengaturan akun orang tua.
			</p>
		</div>
	</article>

	<article class="card space-y-4 p-6 sm:p-8">
		<h2 class="text-xl font-bold flex items-center gap-2">
			<CreditCard size={24} class="text-primary-deep" aria-hidden="true" />
			3. Paket Layanan dan Langganan
		</h2>
		<div class="space-y-3 text-ink-soft leading-relaxed text-sm">
			<p>
				<strong>Paket Gratis:</strong> Menyediakan akses ke materi Kurikulum Merdeka untuk 1 profil anak dengan kuota tantangan harian gratis.
			</p>
			<p>
				<strong>Langganan EduPlay Plus:</strong> Menyediakan profil hingga 5 anak, tantangan harian tanpa batas kuota, laporan perkembangan mendalam, dan rekomendasi latihan adaptif.
			</p>
			<p>
				Pembayaran diproses melalui payment gateway terverifikasi resmi (seperti Midtrans). Anda dapat membatalkan langganan kapan saja melalui menu akun orang tua sebelum tanggal perpanjangan berikutnya.
			</p>
		</div>
	</article>

	<article class="card space-y-4 p-6 sm:p-8">
		<h2 class="text-xl font-bold flex items-center gap-2">
			<WarningCircle size={24} class="text-primary-deep" aria-hidden="true" />
			4. Larangan Penggunaan
		</h2>
		<div class="space-y-2 text-ink-soft leading-relaxed text-sm">
			<p>Pengguna dilarang:</p>
			<ul class="list-disc pl-5 space-y-1">
				<li>Melakukan scraping massal, peretasan, atau dekompilasi terhadap sistem dan basis data EduPlay.</li>
				<li>Menyalahgunakan API atau memalsukan identitas request untuk membypass kuota layanan.</li>
				<li>Menjual kembali akses akun kepada pihak ketiga tanpa izin tertulis dari EduPlay.</li>
			</ul>
		</div>
	</article>

	<article class="card space-y-4 p-6 sm:p-8">
		<h2 class="text-xl font-bold text-ink">5. Hak Kekayaan Intelektual</h2>
		<p class="text-ink-soft leading-relaxed text-sm">
			Seluruh materi pembelajaran, teks soal, sistem penilaian, ilustrasi, suara, dan kode perangkat lunak adalah milik EduPlay atau pemegang lisensi resminya. Penggunaan hanya diizinkan untuk kebutuhan belajar pribadi anak di rumah.
		</p>
	</article>

	<article class="card space-y-4 p-6 sm:p-8">
		<h2 class="text-xl font-bold text-ink">6. Hukum yang Berlaku</h2>
		<p class="text-ink-soft leading-relaxed text-sm">
			Syarat dan ketentuan ini diatur dan ditafsirkan sesuai dengan hukum Negara Kesatuan Republik Indonesia. Setiap perselisihan yang timbul akan diselesaikan secara musyawarah untuk mufakat.
		</p>
	</article>
</div>

<CtaBand title="Mulai Belajar Bersama EduPlay" body="Daftarkan keluarga Anda hari ini dan dampingi tumbuh kembang anak dengan menyenangkan." />
