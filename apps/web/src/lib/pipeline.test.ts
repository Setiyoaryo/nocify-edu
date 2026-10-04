import { describe, expect, it } from 'vitest';
import { hasActiveJob, isActive, jobProgressLabel, type PipelineJob } from './pipeline';

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

	it('falls back to a generic label when there is no order yet', () => {
		expect(jobProgressLabel(job({ status: 'running', progress: {} }))).toBe('Berjalan');
	});
});
