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
	generate: 'AI',
	set_source_status: 'Status sumber',
	watch: 'Pantau kurikulum'
};

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

/** A job still occupying the worker (drives the polling loop). */
export function isActive(job: Pick<PipelineJob, 'status'>): boolean {
	return job.status === 'queued' || job.status === 'running';
}

/** Whether any job in the list is still running (so the page keeps polling). */
export function hasActiveJob(jobs: Pick<PipelineJob, 'status'>[]): boolean {
	return jobs.some(isActive);
}

/** Short human label for a job's progress. */
export function jobProgressLabel(job: PipelineJob): string {
	if (job.status === 'done') return 'Selesai';
	if (job.status === 'failed') return 'Gagal';
	if (job.status === 'cancelled') return 'Dibatalkan';
	if (job.status === 'queued') return 'Menunggu';
	const order = job.progress?.order ?? [];
	const done = order.filter((s) => job.progress?.stages && s in job.progress.stages).length;
	const current = stageLabel[job.progress?.current ?? ''] ?? job.progress?.current ?? '';
	return order.length ? `${done}/${order.length}${current ? ` · ${current}` : ''}` : 'Berjalan';
}
