<script lang="ts">
	import ArrowRight from 'phosphor-svelte/lib/ArrowRight';
	import BookOpen from 'phosphor-svelte/lib/BookOpen';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	import Files from 'phosphor-svelte/lib/Files';
	import Globe from 'phosphor-svelte/lib/Globe';
	import Question from 'phosphor-svelte/lib/Question';
	import Sparkle from 'phosphor-svelte/lib/Sparkle';
	import type { JobOutputMetrics, PipelineMetrics } from '$lib/pipeline';

	let {
		metrics,
		jobMetrics,
		title = 'Ringkasan Hasil untuk Operator'
	}: {
		metrics?: PipelineMetrics;
		jobMetrics?: JobOutputMetrics;
		title?: string;
	} = $props();

	const crawledCount = $derived(jobMetrics ? jobMetrics.pagesFetched : (metrics?.crawled_items ?? 0));
	const chunksCount = $derived(jobMetrics ? jobMetrics.chunksExtracted : (metrics?.extracted_chunks ?? 0));
	const materialsCount = $derived(jobMetrics ? jobMetrics.materialsGenerated : (metrics?.material_candidates ?? 0));
	const questionsCount = $derived(jobMetrics ? jobMetrics.questionsGenerated : (metrics?.question_candidates ?? 0));
	const approvedCount = $derived(metrics?.approved_questions ?? 0);
</script>

<div class="card p-5 sm:p-6" data-testid="pipeline-metrics-card">
	<div class="flex flex-wrap items-center justify-between gap-3 border-b border-line pb-4">
		<div>
			<h3 class="text-xl font-bold">{title}</h3>
			<p class="text-sm text-ink-soft">
				{#if jobMetrics}
					Hasil konkrit yang dihasilkan oleh eksekusi job ini ke dalam sistem.
				{:else}
					Akumulasi materi, potongan teks, dan kandidat soal yang tersedia di staging.
				{/if}
			</p>
		</div>

		<div class="flex flex-wrap items-center gap-2">
			<a href="/app/admin/review?kind=question" class="btn btn-primary btn-sm">
				<Question size={16} weight="bold" aria-hidden="true" />
				Tinjau Soal di Antrian Review
				<ArrowRight size={14} weight="bold" aria-hidden="true" />
			</a>
			<a href="/app/admin/review?kind=material" class="btn btn-quiet btn-sm">
				<BookOpen size={16} weight="bold" aria-hidden="true" />
				Tinjau Materi
			</a>
		</div>
	</div>

	<!-- 4 Core Metrics Grid -->
	<div class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4 sm:gap-4">
		<!-- Crawl -->
		<div class="rounded-2xl border border-line bg-paper-deep/40 p-4 transition-all">
			<div class="flex items-center gap-2 text-ink-soft">
				<Globe size={18} class="text-[#1a6fa3]" aria-hidden="true" />
				<span class="text-xs font-semibold">Dokumen Di-crawl</span>
			</div>
			<div class="mt-2 text-2xl sm:text-3xl font-black text-ink">
				{crawledCount}
			</div>
			<div class="mt-1 text-xs text-ink-soft">
				{#if jobMetrics && metrics?.crawled_items}
					<span class="block">Halaman diambil job</span>
					<span class="text-[11px] opacity-80">(Total sistem: {metrics.crawled_items})</span>
				{:else if jobMetrics}
					Halaman berhasil diambil
				{:else}
					Item sumber di database
				{/if}
			</div>
		</div>

		<!-- Chunks -->
		<div class="rounded-2xl border border-line bg-paper-deep/40 p-4 transition-all">
			<div class="flex items-center gap-2 text-ink-soft">
				<Files size={18} class="text-primary-deep" aria-hidden="true" />
				<span class="text-xs font-semibold">Chunk Teks</span>
			</div>
			<div class="mt-2 text-2xl sm:text-3xl font-black text-ink">
				{chunksCount}
			</div>
			<div class="mt-1 text-xs text-ink-soft">
				{#if jobMetrics && metrics?.extracted_chunks}
					<span class="block">Potongan teks job</span>
					<span class="text-[11px] opacity-80">(Total sistem: {metrics.extracted_chunks})</span>
				{:else}
					Potongan teks terindeks
				{/if}
			</div>
		</div>

		<!-- Materials -->
		<div class="rounded-2xl border border-line bg-paper-deep/40 p-4 transition-all">
			<div class="flex items-center gap-2 text-ink-soft">
				<BookOpen size={18} class="text-sun-deep" aria-hidden="true" />
				<span class="text-xs font-semibold">Materi Terbentuk</span>
			</div>
			<div class="mt-2 text-2xl sm:text-3xl font-black text-ink">
				{materialsCount}
			</div>
			<div class="mt-1 text-xs text-ink-soft">
				{#if materialsCount > 0}
					<span class="font-semibold text-leaf-deep">✓ Materi masuk ke staging</span>
					{#if jobMetrics && metrics?.material_candidates}
						<span class="block text-[11px] opacity-80">(Total sistem: {metrics.material_candidates})</span>
					{/if}
				{:else}
					Belum ada draf materi
				{/if}
			</div>
		</div>

		<!-- Questions -->
		<div class="rounded-2xl border border-line bg-paper-deep/40 p-4 transition-all">
			<div class="flex items-center gap-2 text-ink-soft">
				<Question size={18} class="text-leaf-deep" aria-hidden="true" />
				<span class="text-xs font-semibold">Kandidat Soal</span>
			</div>
			<div class="mt-2 text-2xl sm:text-3xl font-black text-leaf-deep">
				{questionsCount}
			</div>
			<div class="mt-1 text-xs text-ink-soft">
				{#if questionsCount > 0}
					<span class="font-semibold text-leaf-deep">✓ Soal berhasil dibuat!</span>
					{#if jobMetrics && metrics?.question_candidates}
						<span class="block text-[11px] opacity-80">(Total sistem: {metrics.question_candidates})</span>
					{/if}
				{:else}
					Belum ada kandidat soal
				{/if}
				{#if approvedCount > 0 && !jobMetrics}
					<span class="block text-[11px] text-ink-soft">({approvedCount} disetujui)</span>
				{/if}
			</div>
		</div>
	</div>

	<!-- Source Breakdown List (if available from job) -->
	{#if jobMetrics?.sources && jobMetrics.sources.length > 0}
		<div class="mt-4 rounded-xl border border-line bg-paper/60 p-3.5 text-xs">
			<div class="font-bold text-ink mb-2">Rincian Materi & Soal per Sumber:</div>
			<div class="space-y-1.5">
				{#each jobMetrics.sources as src (src.sourceCode)}
					<div class="flex flex-wrap items-center justify-between gap-2 rounded-lg bg-paper-deep/50 px-3 py-1.5">
						<span class="font-mono font-bold text-ink">{src.sourceCode}</span>
						<div class="flex items-center gap-3 text-ink-soft">
							{#if src.pagesFetched > 0}<span>{src.pagesFetched} halaman</span>{/if}
							{#if src.chunksExtracted > 0}<span>{src.chunksExtracted} chunk</span>{/if}
							<span class="font-bold text-ink">{src.materialsGenerated} materi</span>
							<span class="font-bold text-leaf-deep">✓ {src.questionsGenerated} soal</span>
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Status Note for Operator -->
	<div class="mt-4 rounded-xl bg-paper-deep/60 px-4 py-3 text-xs text-ink-soft flex items-center justify-between flex-wrap gap-2">
		<div class="flex items-center gap-2">
			<Sparkle size={16} class="text-primary-deep shrink-0" aria-hidden="true" />
			<span>
				{#if questionsCount > 0 && materialsCount > 0}
					<b>Kesimpulan Operator:</b> Materi dan soal telah <b>benar-benar masuk ke sistem</b>. Siap ditinjau oleh Reviewer di <b>Antrian Review</b>.
				{:else if questionsCount > 0}
					<b>Kesimpulan Operator:</b> {questionsCount} kandidat soal berhasil digenerasi dan siap divalidasi di <b>Antrian Review</b>.
				{:else}
					<b>Kesimpulan Operator:</b> Belum ada soal/materi yang terbentuk. Jalankan pipeline di atas untuk memproses sumber baru.
				{/if}
			</span>
		</div>
		<div class="flex items-center gap-3">
			<a href="/app/admin/review?kind=question" class="font-bold text-primary-deep hover:underline">
				Buka Antrian Review &rarr;
			</a>
		</div>
	</div>
</div>
