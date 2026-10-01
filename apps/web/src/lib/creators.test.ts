import { describe, expect, it } from 'vitest';
import { completionRate, creatorStatusLabel, kindLabel, qualityBand, qualityLabel, shareLabel } from './creators';

describe('creators', () => {
	it('labels kinds and statuses', () => {
		expect(kindLabel('system')).toBe('Sistem');
		expect(kindLabel('staff')).toBe('Internal');
		expect(kindLabel('external')).toBe('Eksternal');
		expect(kindLabel('other')).toBe('other');
		expect(creatorStatusLabel('active')).toBe('Aktif');
		expect(creatorStatusLabel('inactive')).toBe('Nonaktif');
	});

	it('classifies quality bands at the boundaries', () => {
		expect(qualityBand(0)).toBe('none');
		expect(qualityBand(49.99)).toBe('weak');
		expect(qualityBand(50)).toBe('fair');
		expect(qualityBand(69.99)).toBe('fair');
		expect(qualityBand(70)).toBe('good');
		expect(qualityBand(85)).toBe('excellent');
		expect(qualityBand(100)).toBe('excellent');
		expect(qualityLabel(0)).toBe('Belum ada data');
		expect(qualityLabel(90)).toBe('Sangat baik');
	});

	it('formats share percentages', () => {
		expect(shareLabel(0)).toBe('0%');
		expect(shareLabel(12.5)).toBe('12,5%');
		expect(shareLabel(100)).toBe('100%');
	});

	it('computes completion rate', () => {
		expect(completionRate(0, 0)).toBe(0);
		expect(completionRate(4, 2)).toBe(50);
		expect(completionRate(3, 3)).toBe(100);
	});
});
