import { describe, expect, it } from 'vitest';
import data from '../../../../packages/figures/figures.json';
import { figures, palette } from './figures';

// packages/figures/figures.json is the single source of truth shared with the Go
// API and the Python pipeline. This test keeps the web renderer in step with it.
describe('figures', () => {
	const approved = data as { name: string; label: string; color: string }[];

	it('renders every approved question figure with matching label and colour', () => {
		for (const f of approved) {
			const got = figures[f.name];
			expect(got, `missing figure ${f.name}`).toBeDefined();
			expect(got.label, `label for ${f.name}`).toBe(f.label);
			expect(got.color, `colour for ${f.name}`).toBe(f.color);
		}
	});

	it('uses a known palette colour for every figure', () => {
		for (const [name, f] of Object.entries(figures)) {
			expect(palette[f.color], `${name} uses unknown colour ${f.color}`).toBeDefined();
		}
	});

	it('has a fallback figure', () => {
		expect(figures['question']).toBeDefined();
	});
});
