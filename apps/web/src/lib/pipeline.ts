// Types and labels for the pipeline control UI (M11).

export type JobStatus = 'queued' | 'running' | 'done' | 'failed' | 'cancelled';

export type PipelineJob = {
	id: number;
	kind: string;
	params: Record<string, any>;
	status: JobStatus;
	progress: { order?: string[]; current?: string; stages?: Record<string, any> };
	error: string;
	attempts: number;
	max_attempts: number;
	created_at: string;
	started_at: string | null;
	finished_at: string | null;
};

export type PipelineSource = {
	code: string;
	name: string;
	status: string;
	enabled: boolean;
	crawl_frequency: string;
	base_url: string;
	last_crawled_at: string | null;
	pending_jobs: number;
};

export type WorkerStatus = 'active' | 'idle' | 'stalled' | 'stopped';

export type WorkerInfo = {
	status: WorkerStatus;
	label: string;
	last_heartbeat_at: string | null;
	current_job_id?: number | null;
	running_jobs: number;
	queued_jobs: number;
	message: string;
};

export type PipelineMetrics = {
	crawled_items: number;
	extracted_chunks: number;
	material_candidates: number;
	question_candidates: number;
	approved_questions: number;
};

export const JOB_KINDS = ['run_pipeline', 'crawl', 'watch', 'set_source_status'] as const;
export const CRAWL_FREQUENCIES = ['daily', 'weekly', 'monthly', 'manual'] as const;
export const SOURCE_STATUSES = ['discovered', 'verified', 'approved', 'active', 'paused', 'retired'] as const;

export const jobKindLabel: Record<string, string> = {
	run_pipeline: 'Jalankan pipeline',
	crawl: 'Crawl sumber',
	watch: 'Pantau kurikulum',
	set_source_status: 'Ubah status sumber'
};

export const stageLabel: Record<string, string> = {
	crawl: 'Crawl',
	extract: 'Ekstrak',
	embed: 'Embedding',
	map: 'Pemetaan',
	materials: 'Materi',
	questions: 'Soal',
	visuals: 'Visual',
	generate: 'AI Generasi',
	set_source_status: 'Status sumber',
	watch: 'Pantau kurikulum'
};

export const stageDescription: Record<string, string> = {
	crawl: 'Mengambil konten dari sumber resmi',
	extract: 'Mengekstrak teks & memecah jadi chunk',
	embed: 'Menghitung representasi vektor semantik',
	map: 'Memetakan materi ke kurikulum & CP',
	materials: 'Menyusun draf materi pembelajaran',
	questions: 'Membuat kandidat soal asesmen',
	visuals: 'Memetakan ilustrasi visual & diagram',
	generate: 'Generasi soal berbantuan LLM AI',
	set_source_status: 'Memperbarui konfigurasi sumber',
	watch: 'Memantau perubahan kurikulum'
};

export const DEFAULT_PIPELINE_STAGES = [
	'crawl',
	'extract',
	'map',
	'materials',
	'questions',
	'visuals'
] as const;

export const frequencyLabel: Record<string, string> = {
	daily: 'Harian',
	weekly: 'Mingguan',
	monthly: 'Bulanan',
	manual: 'Manual'
};

export const sourceStatusLabel: Record<string, string> = {
	discovered: 'Ditemukan',
	verified: 'Terverifikasi',
	approved: 'Disetujui',
	active: 'Aktif',
	paused: 'Dijeda',
	retired: 'Diarsipkan'
};

export const workerStatusStyle: Record<WorkerStatus, { badge: string; dot: string; label: string }> = {
	active: { badge: 'bg-leaf-tint text-leaf-deep border-leaf-tint', dot: 'bg-leaf-deep animate-ping', label: 'Aktif' },
	idle: { badge: 'bg-sky-tint text-[#1a6fa3] border-sky-tint', dot: 'bg-[#1a6fa3]', label: 'Siaga (Idle)' },
	stalled: { badge: 'bg-sun-tint text-ink border-sun-tint', dot: 'bg-sun-deep animate-pulse', label: 'Terhenti' },
	stopped: { badge: 'bg-rose-tint text-[#a8264f] border-rose-tint', dot: 'bg-[#a8264f]', label: 'Offline' }
};

/** A job still occupying the worker (drives the polling loop). */
export function isActive(job: Pick<PipelineJob, 'status'>): boolean {
	return job.status === 'queued' || job.status === 'running';
}

/** Whether any job in the list is still running (so the page keeps polling). */
export function hasActiveJob(jobs: Pick<PipelineJob, 'status'>[]): boolean {
	return jobs.some(isActive);
}

/** Returns the list of stages to display for this job in execution order. */
export function getJobStages(job: PipelineJob): string[] {
	if (job.progress?.order && job.progress.order.length > 0) {
		return job.progress.order;
	}
	if (job.kind === 'run_pipeline') {
		const stages = ((job.params?.stages as string[]) || [...DEFAULT_PIPELINE_STAGES]).slice();
		if (job.params?.ai) {
			if (!stages.includes('embed') && stages.includes('map')) {
				stages.splice(stages.indexOf('map'), 0, 'embed');
			}
			if (!stages.includes('generate') && stages.includes('questions')) {
				stages.splice(stages.indexOf('questions') + 1, 0, 'generate');
			}
		}
		return stages;
	}
	if (job.kind === 'crawl') return ['crawl'];
	if (job.kind === 'watch') return ['watch'];
	if (job.kind === 'set_source_status') return ['set_source_status'];
	return [job.kind];
}

export type StageStepStatus = 'done' | 'running' | 'queued' | 'failed' | 'cancelled';

/** Status of an individual stage step in a pipeline job. */
export function getStageStepStatus(job: PipelineJob, stage: string): StageStepStatus {
	const stages = job.progress?.stages ?? {};
	if (stage in stages) return 'done';
	if (job.status === 'done') return 'done';

	const order = getJobStages(job);
	const doneCount = order.filter((s) => s in stages).length;

	if (job.status === 'running') {
		if (job.progress?.current === stage) return 'running';
		const currentIdx = order.indexOf(job.progress?.current ?? '');
		const stageIdx = order.indexOf(stage);
		if (currentIdx !== -1 && stageIdx !== -1 && stageIdx < currentIdx) return 'done';
		if (!job.progress?.current && order.indexOf(stage) === doneCount) return 'running';
		return 'queued';
	}
	if (job.status === 'failed') {
		const failedStage = job.progress?.current || order[doneCount] || order[0] || '';
		if (stage === failedStage) return 'failed';
		const failedIdx = order.indexOf(failedStage);
		const stageIdx = order.indexOf(stage);
		if (failedIdx !== -1 && stageIdx !== -1 && stageIdx < failedIdx) return 'done';
		return 'queued';
	}
	if (job.status === 'cancelled') {
		const cancelledStage = job.progress?.current || order[doneCount] || '';
		if (stage === cancelledStage) return 'cancelled';
		return 'queued';
	}
	return 'queued';
}

/** Detailed progress computation for progress bars and stage trackers. */
export function calculateJobProgress(job: PipelineJob): {
	percent: number;
	doneCount: number;
	totalCount: number;
	stepNumber: number;
	currentStage: string;
	currentStageLabel: string;
	currentStageDesc: string;
} {
	const order = getJobStages(job);
	const totalCount = Math.max(order.length, 1);
	const doneCount = order.filter((s) => job.progress?.stages && s in job.progress.stages).length;
	const currentStage = job.progress?.current ?? (job.status === 'running' ? order[doneCount] ?? order[totalCount - 1] ?? '' : '');
	const currentLabel = stageLabel[currentStage] ?? currentStage;
	const currentDesc = stageDescription[currentStage] ?? '';

	const stageIdx = order.indexOf(currentStage);
	const stepNumber = stageIdx !== -1 ? Math.min(stageIdx + 1, totalCount) : Math.min(doneCount + 1, totalCount);

	if (job.status === 'done') {
		return {
			percent: 100,
			doneCount: totalCount,
			totalCount,
			stepNumber: totalCount,
			currentStage: '',
			currentStageLabel: 'Selesai',
			currentStageDesc: 'Semua tahapan berhasil dijalankan'
		};
	}
	if (job.status === 'queued') {
		return {
			percent: 0,
			doneCount: 0,
			totalCount,
			stepNumber: 1,
			currentStage: order[0] ?? '',
			currentStageLabel: 'Menunggu',
			currentStageDesc: 'Job berada dalam antrean eksekusi worker'
		};
	}
	if (job.status === 'failed') {
		return {
			percent: Math.round((doneCount / totalCount) * 100),
			doneCount,
			totalCount,
			stepNumber,
			currentStage,
			currentStageLabel: 'Gagal',
			currentStageDesc: job.error || 'Terjadi kesalahan pada tahapan ini'
		};
	}
	if (job.status === 'cancelled') {
		return {
			percent: Math.round((doneCount / totalCount) * 100),
			doneCount,
			totalCount,
			stepNumber,
			currentStage,
			currentStageLabel: 'Dibatalkan',
			currentStageDesc: 'Job dibatalkan oleh operator'
		};
	}

	// Running
	const effectiveDone = Math.min(doneCount + (currentStage ? 0.5 : 0), totalCount);
	const percent = Math.min(95, Math.max(5, Math.round((effectiveDone / totalCount) * 100)));
	return {
		percent,
		doneCount,
		totalCount,
		stepNumber,
		currentStage,
		currentStageLabel: currentLabel || 'Berjalan',
		currentStageDesc: currentDesc || 'Sedang memproses data'
	};
}

export type SourceMetricSummary = {
	sourceCode: string;
	pagesDiscovered: number;
	pagesFetched: number;
	chunksExtracted: number;
	materialsGenerated: number;
	materialsDrafts: number;
	questionsGenerated: number;
	questionsDrafts: number;
	visualsProposed: number;
};

export type JobOutputMetrics = {
	pagesDiscovered: number;
	pagesFetched: number;
	chunksExtracted: number;
	materialsGenerated: number;
	materialsDrafts: number;
	questionsGenerated: number;
	questionsDrafts: number;
	visualsProposed: number;
	hasResults: boolean;
	sources: SourceMetricSummary[];
};

function normalizeStageItems(val: any): any[] {
	if (!val) return [];
	if (Array.isArray(val)) return val;
	if (typeof val === 'object') return [val];
	return [];
}

/** Extracts structured counts from job.progress.stages for operator summary cards. */
export function extractJobMetrics(job: PipelineJob): JobOutputMetrics {
	const stages = job.progress?.stages ?? {};
	let pagesDiscovered = 0;
	let pagesFetched = 0;
	let chunksExtracted = 0;
	let materialsGenerated = 0;
	let materialsDrafts = 0;
	let questionsGenerated = 0;
	let questionsDrafts = 0;
	let visualsProposed = 0;

	const sourceMap = new Map<string, SourceMetricSummary>();
	const getSource = (code: string): SourceMetricSummary => {
		const key = code || 'umum';
		if (!sourceMap.has(key)) {
			sourceMap.set(key, {
				sourceCode: key,
				pagesDiscovered: 0,
				pagesFetched: 0,
				chunksExtracted: 0,
				materialsGenerated: 0,
				materialsDrafts: 0,
				questionsGenerated: 0,
				questionsDrafts: 0,
				visualsProposed: 0
			});
		}
		return sourceMap.get(key)!;
	};

	// Crawl stage
	for (const item of normalizeStageItems(stages.crawl)) {
		const disc = item.pages_discovered ?? 0;
		const fetch = item.pages_fetched ?? 0;
		pagesDiscovered += disc;
		pagesFetched += fetch;
		if (item.source_code) {
			const s = getSource(item.source_code);
			s.pagesDiscovered += disc;
			s.pagesFetched += fetch;
		}
	}

	// Extract stage
	for (const item of normalizeStageItems(stages.extract)) {
		const chk = item.chunks ?? 0;
		chunksExtracted += chk;
		if (item.source_code) {
			const s = getSource(item.source_code);
			s.chunksExtracted += chk;
		}
	}

	// Materials stage
	for (const item of normalizeStageItems(stages.materials)) {
		const gen = item.generated ?? 0;
		const drf = item.drafts ?? 0;
		materialsGenerated += gen;
		materialsDrafts += drf;
		if (item.source_code) {
			const s = getSource(item.source_code);
			s.materialsGenerated += gen;
			s.materialsDrafts += drf;
		}
	}

	// Questions stage
	for (const item of normalizeStageItems(stages.questions)) {
		const gen = item.generated ?? 0;
		const drf = item.drafts ?? 0;
		questionsGenerated += gen;
		questionsDrafts += drf;
		if (item.source_code) {
			const s = getSource(item.source_code);
			s.questionsGenerated += gen;
			s.questionsDrafts += drf;
		}
	}

	// AI Generate stage
	for (const item of normalizeStageItems(stages.generate)) {
		const gen = item.generated ?? 0;
		questionsGenerated += gen;
		if (item.source_code) {
			const s = getSource(item.source_code);
			s.questionsGenerated += gen;
		}
	}

	// Visuals stage
	for (const item of normalizeStageItems(stages.visuals)) {
		const prop = item.proposed ?? 0;
		visualsProposed += prop;
		if (item.source_code) {
			const s = getSource(item.source_code);
			s.visualsProposed += prop;
		}
	}

	const hasResults =
		pagesFetched > 0 ||
		chunksExtracted > 0 ||
		materialsGenerated > 0 ||
		questionsGenerated > 0 ||
		visualsProposed > 0;

	return {
		pagesDiscovered,
		pagesFetched,
		chunksExtracted,
		materialsGenerated,
		materialsDrafts,
		questionsGenerated,
		questionsDrafts,
		visualsProposed,
		hasResults,
		sources: Array.from(sourceMap.values())
	};
}

/** Formats elapsed duration between started_at and finished_at (or now). */
export function formatDuration(startedAt: string | null, finishedAt: string | null): string {
	if (!startedAt) return '—';
	const start = new Date(startedAt).getTime();
	if (isNaN(start)) return '—';
	const end = finishedAt ? new Date(finishedAt).getTime() : Date.now();
	const diffMs = Math.max(0, end - start);
	const diffSec = Math.floor(diffMs / 1000);
	if (diffSec < 60) return `${diffSec} dtk`;
	const mins = Math.floor(diffSec / 60);
	const remSec = diffSec % 60;
	if (mins < 60) return `${mins} mnt ${remSec} dtk`;
	const hrs = Math.floor(mins / 60);
	const remMins = mins % 60;
	return `${hrs} jam ${remMins} mnt`;
}

/** Short human label for a job's progress. */
export function jobProgressLabel(job: PipelineJob): string {
	if (job.status === 'done') return 'Selesai';
	if (job.status === 'failed') return 'Gagal';
	if (job.status === 'cancelled') return 'Dibatalkan';
	if (job.status === 'queued') return 'Menunggu';
	const order = getJobStages(job);
	const done = order.filter((s) => job.progress?.stages && s in job.progress.stages).length;
	const current = stageLabel[job.progress?.current ?? ''] ?? job.progress?.current ?? '';
	return order.length ? `${done}/${order.length}${current ? ` · ${current}` : ''}` : 'Berjalan';
}
