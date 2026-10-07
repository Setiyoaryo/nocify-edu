<script lang="ts">
	import Meta from '$lib/components/Meta.svelte';
	import PageHero from '$lib/components/PageHero.svelte';
	import CtaBand from '$lib/components/CtaBand.svelte';
	import ShieldCheck from 'phosphor-svelte/lib/ShieldCheck';
	import LockKey from 'phosphor-svelte/lib/LockKey';
	import UserCheck from 'phosphor-svelte/lib/UserCheck';
	import Trash from 'phosphor-svelte/lib/Trash';
	import Prohibit from 'phosphor-svelte/lib/Prohibit';
	import SpeakerHigh from 'phosphor-svelte/lib/SpeakerHigh';
	import HardDrive from 'phosphor-svelte/lib/HardDrive';
	import FileText from 'phosphor-svelte/lib/FileText';
</script>

<Meta
	title="Kebijakan Privasi & Perlindungan Data Anak"
	description="Komitmen perlindungan privasi dan data anak di EduPlay sesuai UU PDP No. 27/2022, PP 17/2025, dan Permenkomdigi 9/2026."
/>

<PageHero
	title="Kebijakan Privasi & Perlindungan Data Anak"
	lead="Di EduPlay, data anak dilindungi secara ketat. Kami merancang platform ini dengan prinsip privasi sejak awal (Privacy by Design), berlandaskan regulasi perlindungan data pribadi Republik Indonesia."
/>

<div class="mx-auto max-w-4xl space-y-10 px-4 sm:px-6">
	<div class="rounded-2xl border border-primary/20 bg-primary-tint/30 p-5 sm:p-6 text-ink">
		<div class="flex items-start gap-3">
			<ShieldCheck size={28} weight="duotone" class="shrink-0 text-primary-deep" aria-hidden="true" />
			<div>
				<h2 class="text-lg font-bold">Kepatuhan Regulasi</h2>
				<p class="mt-1 text-sm text-ink-soft">
					Kebijakan privasi ini disusun memenuhi ketentuan Undang-Undang No. 27 Tahun 2022 tentang Perlindungan Data Pribadi (UU PDP),
					Peraturan Pemerintah No. 17 Tahun 2025, serta Peraturan Menkomdigi No. 9 Tahun 2026 mengenai pemrosesan data spesifik anak.
				</p>
			</div>
		</div>
	</div>

	<section class="space-y-6">
		<div class="grid gap-6 sm:grid-cols-2">
			<article class="card p-6">
				<div class="flex items-center gap-3">
					<span class="grid size-10 place-items-center rounded-xl bg-primary-tint text-primary-deep">
						<UserCheck size={24} weight="bold" aria-hidden="true" />
					</span>
					<h3 class="text-xl font-bold">Persetujuan Orang Tua (Consent)</h3>
				</div>
				<p class="mt-3 text-ink-soft text-sm leading-relaxed">
					Pendaftaran akun mewajibkan konfirmasi persetujuan orang tua atau wali yang sah. Kami mencatat versi teks persetujuan
					(<code class="rounded bg-paper-deep px-1.5 py-0.5 text-xs">consent_version</code>) dan waktu persetujuan
					(<code class="rounded bg-paper-deep px-1.5 py-0.5 text-xs">consent_at</code>) sebagai bukti persetujuan pemrosesan data anak.
				</p>
			</article>

			<article class="card p-6">
				<div class="flex items-center gap-3">
					<span class="grid size-10 place-items-center rounded-xl bg-leaf-tint text-leaf-deep">
						<FileText size={24} weight="bold" aria-hidden="true" />
					</span>
					<h3 class="text-xl font-bold">Prinsip Data Minimal</h3>
				</div>
				<p class="mt-3 text-ink-soft text-sm leading-relaxed">
					Profil anak hanya menyimpan nama panggilan (nickname), jenjang kelas, avatar kartun pilihan, opsi 4-angka PIN (terenkripsi bcrypt),
					dan rekam jejak capaian latihan belajar. Kami <strong>tidak pernah</strong> meminta tanggal lahir, foto asli, alamat rumah, nomor telepon, nama sekolah, maupun lokasi GPS.
				</p>
			</article>

			<article class="card p-6">
				<div class="flex items-center gap-3">
					<span class="grid size-10 place-items-center rounded-xl bg-sun-tint text-tangerine-deep">
						<LockKey size={24} weight="bold" aria-hidden="true" />
					</span>
					<h3 class="text-xl font-bold">Isolasi Akun Orang Tua</h3>
				</div>
				<p class="mt-3 text-ink-soft text-sm leading-relaxed">
					Setiap permintaan data anak dan riwayat challenge selalu dibatasi ketat oleh identitas akun orang tua (<code class="rounded bg-paper-deep px-1.5 py-0.5 text-xs">parent_id</code>).
					Tidak ada profil publik, tidak ada leaderboard global antaranak, dan pengguna lain akan selalu menerima respon 404 jika mencoba mengakses data di luar keluarganya.
				</p>
			</article>

			<article class="card p-6">
				<div class="flex items-center gap-3">
					<span class="grid size-10 place-items-center rounded-xl bg-punch-tint text-berry-deep">
						<Trash size={24} weight="bold" aria-hidden="true" />
					</span>
					<h3 class="text-xl font-bold">Hak Penghapusan Data (Erasure)</h3>
				</div>
				<p class="mt-3 text-ink-soft text-sm leading-relaxed">
					Orang tua berhak menghapus profil anak atau seluruh akun kapan pun melalui menu pengaturan. Proses penghapusan langsung menghapus
					secara berantai (cascade) seluruh riwayat pengerjaan, jawaban soal, progres skill, dan inventaris hadiah.
				</p>
			</article>

			<article class="card p-6">
				<div class="flex items-center gap-3">
					<span class="grid size-10 place-items-center rounded-xl bg-paper-deep text-ink">
						<Prohibit size={24} weight="bold" aria-hidden="true" />
					</span>
					<h3 class="text-xl font-bold">Tanpa Iklan & Tanpa Profiling</h3>
				</div>
				<p class="mt-3 text-ink-soft text-sm leading-relaxed">
					EduPlay 100% bebas dari jaringan iklan komersial maupun pelacak pihak ketiga (third-party tracking). Analitik hanya berjalan internal
					(first-party) dengan daftar event terbatas untuk mengevaluasi kualitas pedagogis materi, bukan untuk profiling komersial anak.
				</p>
			</article>

			<article class="card p-6">
				<div class="flex items-center gap-3">
					<span class="grid size-10 place-items-center rounded-xl bg-mint-tint text-leaf-deep">
						<HardDrive size={24} weight="bold" aria-hidden="true" />
					</span>
					<h3 class="text-xl font-bold">Aset Mandiri & Log Aman</h3>
				</div>
				<p class="mt-3 text-ink-soft text-sm leading-relaxed">
					Font dan ilustrasi disajikan langsung dari infrastruktur mandiri kami (<code class="rounded bg-paper-deep px-1.5 py-0.5 text-xs">@fontsource</code>).
					Log server tidak pernah mencatat isi body permintaan yang berisi jawaban anak atau data sensitif, dan metrik latensi hanya mengukur pola rute.
				</p>
			</article>
		</div>

		<article class="card p-6">
			<div class="flex items-center gap-3">
				<span class="grid size-10 place-items-center rounded-xl bg-primary-tint text-primary-deep">
					<SpeakerHigh size={24} weight="bold" aria-hidden="true" />
				</span>
				<h3 class="text-xl font-bold">Sintesis Suara Berjalan di Perangkat (Lokal)</h3>
			</div>
			<p class="mt-3 text-ink-soft text-sm leading-relaxed">
				Fitur bantuan pembacaan soal menggunakan Web Speech API bawaan browser anak secara offline di perangkat pengguna. Tidak ada transmisi suara atau rekaman audio yang dikirim ke server kami maupun layanan kecerdasan buatan luar saat anak mendengarkan soal.
			</p>
		</article>
	</section>

	<section class="card space-y-4 p-6 sm:p-8">
		<h2 class="text-2xl font-bold">Hak Anda Sebagai Orang Tua</h2>
		<p class="text-ink-soft leading-relaxed">
			Sesuai UU PDP, Anda memiliki hak penuh untuk mengakses data anak Anda, memperbarui data profil, mengatur PIN pengaman akses anak,
			serta meminta penghapusan permanen akun dan seluruh riwayat belajarnya. Jika Anda memiliki pertanyaan seputar perlindungan privasi anak di EduPlay,
			Anda dapat menghubungi Petugas Perlindungan Data kami melalui dashboard orang tua.
		</p>
	</section>
</div>

<CtaBand title="Eksplorasi EduPlay dengan Rasa Aman" body="Coba tantangan kurikulum Merdeka untuk anak Anda hari ini. Gratis untuk 1 profil anak tanpa kartu kredit." />
