import { describe, expect, it } from 'vitest';
import { relative, rupiah } from './format';

describe('format', () => {
	it('formats rupiah with Indonesian grouping', () => {
		expect(rupiah(49000)).toBe('Rp49.000');
	});
	it('handles missing activity', () => {
		expect(relative(null)).toBe('Belum pernah');
	});
	it('describes recent activity', () => {
		expect(relative(new Date().toISOString())).toBe('Baru saja');
	});
});
