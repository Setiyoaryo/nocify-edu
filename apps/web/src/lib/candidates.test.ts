import { describe, expect, it } from 'vitest';
import { bandFor, blockingCount, confidenceLabel, kindLabel, qaIssues } from './candidates';

describe('candidates', () => {
	it('maps curriculum phase to a preview band', () => {
		expect(bandFor('fondasi')).toBe('early');
		expect(bandFor('a')).toBe('early');
		expect(bandFor('b')).toBe('lower');
		expect(bandFor('c')).toBe('upper');
		expect(bandFor('unknown')).toBe('upper');
	});

	it('labels candidate kinds', () => {
		expect(kindLabel('question')).toBe('Soal');
		expect(kindLabel('material')).toBe('Materi');
	});

	it('normalises QA issues', () => {
		expect(qaIssues(null)).toEqual([]);
		expect(qaIssues({})).toEqual([]);
		const issues = [
			{ code: 'a', message: 'm', blocking: true },
			{ code: 'b', message: 'm', blocking: false }
		];
		expect(qaIssues(issues)).toHaveLength(2);
		expect(blockingCount(qaIssues(issues))).toBe(1);
	});

	it('renders confidence as a percentage', () => {
		expect(confidenceLabel(0.82)).toBe('82%');
		expect(confidenceLabel(0)).toBe('0%');
		expect(confidenceLabel(1)).toBe('100%');
	});
});
