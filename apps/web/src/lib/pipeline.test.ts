import { describe, expect, it } from 'vitest';
import {
	calculateJobProgress,
	extractJobMetrics,
	formatDuration,
	getJobStages,
	getStageStepStatus,
	hasActiveJob,
	isActive,
	jobProgressLabel,
	type PipelineJob
} from './pipeline';

function job(over: Partial<PipelineJob>): PipelineJob {
	return {
		id: 1,
		kind: 'run_pipeline',
		params: {},
		status: 'queued',
		progress: {},
		error: '',
		attempts: 0,
		max_attempts: 3,
		created_at: '2026-10-04T00:00:00Z',
		started_at: null,
		finished_at: null,
		...over
	};
}

describe('pipeline helpers', () => {
	it('treats queued and running jobs as active', () => {
		expect(isActive(job({ status: 'queued' }))).toBe(true);
		expect(isActive(job({ status: 'running' }))).toBe(true);
		expect(isActive(job({ status: 'done' }))).toBe(false);
		expect(isActive(job({ status: 'failed' }))).toBe(false);
		expect(isActive(job({ status: 'cancelled' }))).toBe(false);
	});

	it('detects any active job in a list', () => {
		expect(hasActiveJob([job({ status: 'done' }), job({ status: 'queued' })])).toBe(true);
		expect(hasActiveJob([job({ status: 'done' }), job({ status: 'failed' })])).toBe(false);
		expect(hasActiveJob([])).toBe(false);
	});

	it('labels terminal states plainly', () => {
		expect(jobProgressLabel(job({ status: 'done' }))).toBe('Selesai');
		expect(jobProgressLabel(job({ status: 'failed' }))).toBe('Gagal');
		expect(jobProgressLabel(job({ status: 'cancelled' }))).toBe('Dibatalkan');
		expect(jobProgressLabel(job({ status: 'queued' }))).toBe('Menunggu');
	});

	it('summarises running progress from the stage order', () => {
		const running = job({
			status: 'running',
			progress: {
				order: ['crawl', 'extract', 'map'],
				current: 'map',
				stages: { crawl: {}, extract: {} }
			}
		});
		expect(jobProgressLabel(running)).toBe('2/3 · Pemetaan');
	});

	it('falls back to default pipeline stages when order is empty', () => {
		const j = job({ status: 'running', progress: {} });
		expect(jobProgressLabel(j)).toBe('0/6');
		expect(getJobStages(j)).toEqual(['crawl', 'extract', 'map', 'materials', 'questions', 'visuals']);
	});

	it('resolves stages for different job kinds', () => {
		expect(getJobStages(job({ kind: 'crawl' }))).toEqual(['crawl']);
		expect(getJobStages(job({ kind: 'watch' }))).toEqual(['watch']);
		expect(getJobStages(job({ kind: 'set_source_status' }))).toEqual(['set_source_status']);

		// With AI enabled
		const aiJob = job({ kind: 'run_pipeline', params: { ai: true } });
		const stages = getJobStages(aiJob);
		expect(stages).toContain('embed');
		expect(stages).toContain('generate');
		expect(stages.indexOf('embed')).toBeLessThan(stages.indexOf('map'));
	});

	it('determines individual stage step statuses accurately', () => {
		const running = job({
			status: 'running',
			progress: {
				order: ['crawl', 'extract', 'map', 'materials', 'questions', 'visuals'],
				current: 'materials',
				stages: { crawl: {}, extract: {}, map: {} }
			}
		});

		expect(getStageStepStatus(running, 'crawl')).toBe('done');
		expect(getStageStepStatus(running, 'extract')).toBe('done');
		expect(getStageStepStatus(running, 'map')).toBe('done');
		expect(getStageStepStatus(running, 'materials')).toBe('running');
		expect(getStageStepStatus(running, 'questions')).toBe('queued');
		expect(getStageStepStatus(running, 'visuals')).toBe('queued');

		const failed = job({
			status: 'failed',
			error: 'AI timeout',
			progress: {
				order: ['crawl', 'extract', 'map'],
				current: 'map',
				stages: { crawl: {}, extract: {} }
			}
		});
		expect(getStageStepStatus(failed, 'extract')).toBe('done');
		expect(getStageStepStatus(failed, 'map')).toBe('failed');
	});

	it('calculates job progress percentage and labels correctly', () => {
		// Done
		const done = job({ status: 'done' });
		const doneProg = calculateJobProgress(done);
		expect(doneProg.percent).toBe(100);
		expect(doneProg.currentStageLabel).toBe('Selesai');

		// Queued
		const queued = job({ status: 'queued' });
		const queuedProg = calculateJobProgress(queued);
		expect(queuedProg.percent).toBe(0);
		expect(queuedProg.currentStageLabel).toBe('Menunggu');

		// Running midway
		const running = job({
			status: 'running',
			progress: {
				order: ['crawl', 'extract', 'map', 'materials', 'questions', 'visuals'],
				current: 'materials',
				stages: { crawl: {}, extract: {}, map: {} }
			}
		});
		const runningProg = calculateJobProgress(running);
		expect(runningProg.doneCount).toBe(3);
		expect(runningProg.totalCount).toBe(6);
		expect(runningProg.stepNumber).toBe(4);
		// 3.5 / 6 = 58%
		expect(runningProg.percent).toBe(58);
		expect(runningProg.currentStageLabel).toBe('Materi');

		// Running at last stage - stepNumber must not exceed totalCount
		const runningLast = job({
			status: 'running',
			progress: {
				order: ['crawl', 'extract'],
				current: 'extract',
				stages: { crawl: {}, extract: {} }
			}
		});
		const lastProg = calculateJobProgress(runningLast);
		expect(lastProg.stepNumber).toBe(2);
		expect(lastProg.stepNumber).toBeLessThanOrEqual(lastProg.totalCount);

		// Failed
		const failed = job({
			status: 'failed',
			error: 'Koneksi terputus',
			progress: {
				order: ['crawl', 'extract'],
				current: 'extract',
				stages: { crawl: {} }
			}
		});
		const failedProg = calculateJobProgress(failed);
		expect(failedProg.currentStageLabel).toBe('Gagal');
		expect(failedProg.currentStageDesc).toBe('Koneksi terputus');
	});

	it('extracts metrics summary from stages including AI generate and source breakdown', () => {
		const j = job({
			status: 'done',
			progress: {
				stages: {
					crawl: [{ source_code: 'kemendikdasmen-kurikulum', pages_discovered: 10, pages_fetched: 8 }],
					extract: [{ source_code: 'kemendikdasmen-kurikulum', chunks: 25, documents: 5 }],
					materials: [{ source_code: 'kemendikdasmen-kurikulum', generated: 6, drafts: 4 }],
					questions: [{ source_code: 'kemendikdasmen-kurikulum', generated: 12, drafts: 8, needs_review: 4 }],
					generate: { generated: 5 }, // AI generator returned object
					visuals: [{ source_code: 'kemendikdasmen-kurikulum', proposed: 3 }]
				}
			}
		});

		const metrics = extractJobMetrics(j);
		expect(metrics.pagesDiscovered).toBe(10);
		expect(metrics.pagesFetched).toBe(8);
		expect(metrics.chunksExtracted).toBe(25);
		expect(metrics.materialsGenerated).toBe(6);
		expect(metrics.materialsDrafts).toBe(4);
		expect(metrics.questionsGenerated).toBe(17); // 12 + 5 AI generated
		expect(metrics.visualsProposed).toBe(3);
		expect(metrics.hasResults).toBe(true);
		expect(metrics.sources.length).toBeGreaterThanOrEqual(1);
		expect(metrics.sources[0].sourceCode).toBe('kemendikdasmen-kurikulum');
		expect(metrics.sources[0].materialsGenerated).toBe(6);

		// Empty job
		const empty = extractJobMetrics(job({ status: 'queued' }));
		expect(empty.hasResults).toBe(false);
		expect(empty.questionsGenerated).toBe(0);
		expect(empty.sources).toEqual([]);
	});

	it('formats duration nicely', () => {
		expect(formatDuration(null, null)).toBe('—');
		expect(formatDuration('2026-10-04T10:00:00Z', '2026-10-04T10:00:25Z')).toBe('25 dtk');
		expect(formatDuration('2026-10-04T10:00:00Z', '2026-10-04T10:02:15Z')).toBe('2 mnt 15 dtk');
		expect(formatDuration('2026-10-04T10:00:00Z', '2026-10-04T11:15:00Z')).toBe('1 jam 15 mnt');
	});
});
