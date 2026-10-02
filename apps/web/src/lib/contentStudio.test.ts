import { describe, expect, it } from 'vitest';
import { hasAudio, hasVisual, skillHasWork, visualLabel } from './contentStudio';
import type { StudioSkill } from './contentStudio';

function detail(over: Record<string, unknown>) {
	return over as any;
}

describe('contentStudio', () => {
	it('labels a question visual treatment', () => {
		expect(visualLabel(true, false)).toBe('Ada visual');
		expect(visualLabel(false, true)).toBe('Butuh audio');
		expect(visualLabel(false, false)).toBe('Teks saja');
		// Visual wins over audio when both are present.
		expect(visualLabel(true, true)).toBe('Ada visual');
	});

	it('prefers the server has_visual flag', () => {
		expect(hasVisual(detail({ has_visual: true }))).toBe(true);
		expect(hasVisual(detail({ has_visual: false }))).toBe(false);
	});

	it('falls back to content.figures / content.image', () => {
		expect(hasVisual(detail({ payload: { content: { figures: [{ id: 'f1' }] } } }))).toBe(true);
		expect(hasVisual(detail({ payload: { content: { image: '/a.png' } } }))).toBe(true);
		expect(hasVisual(detail({ payload: { content: { figures: [] } } }))).toBe(false);
		expect(hasVisual(detail({ payload: { content: { text: 'halo' } } }))).toBe(false);
	});

	it('detects figures/images nested in image_choice options', () => {
		expect(
			hasVisual(detail({ payload: { content: { options: [{ figures: [{ id: 'f1' }] }, {}] } } }))
		).toBe(true);
		expect(hasVisual(detail({ payload: { content: { options: [{ image: '/b.png' }] } } }))).toBe(true);
		expect(hasVisual(detail({ payload: { content: { options: [{ text: 'a' }, { text: 'b' }] } } }))).toBe(
			false
		);
	});

	it('treats a missing payload as text-only', () => {
		expect(hasVisual(detail({}))).toBe(false);
		expect(hasVisual(detail({ payload: null }))).toBe(false);
		expect(hasVisual(detail({ payload: {} }))).toBe(false);
	});

	it('detects audio from the flag or the payload', () => {
		expect(hasAudio(detail({ has_audio: true }))).toBe(true);
		expect(hasAudio(detail({ has_audio: false }))).toBe(false);
		expect(hasAudio(detail({ payload: { content: { audio_text: 'baca ini' } } }))).toBe(true);
		expect(hasAudio(detail({ payload: { content: { audio_text: '' } } }))).toBe(false);
		expect(hasAudio(detail({}))).toBe(false);
	});

	it('reports whether a skill has any work', () => {
		const base: StudioSkill = {
			id: 1,
			code: 's',
			name: 'Skill',
			materials: 0,
			questions: 0,
			needs_review: 0,
			published: 0
		};
		expect(skillHasWork(base)).toBe(false);
		expect(skillHasWork({ ...base, materials: 1 })).toBe(true);
		expect(skillHasWork({ ...base, questions: 2 })).toBe(true);
		expect(skillHasWork({ ...base, published: 1 })).toBe(true);
	});
});
