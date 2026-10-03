import { describe, expect, it } from 'vitest';
import {
	hasAudio,
	hasVisual,
	needsVisual,
	skillHasWork,
	visualKindLabel,
	visualKindOf,
	visualSummaryOf
} from './contentStudio';
import type { StudioSkill } from './contentStudio';

function detail(over: Record<string, unknown>) {
	return over as any;
}

describe('contentStudio', () => {
	it('labels a question visual treatment', () => {
		expect(visualKindLabel('image')).toBe('Visual: gambar');
		expect(visualKindLabel('icon')).toBe('Visual: ikon');
		expect(visualKindLabel('audio')).toBe('Butuh audio');
		expect(visualKindLabel('text')).toBe('Teks saja');
	});

	it('picks the visual kind from the payload', () => {
		expect(visualKindOf(false, false, { image: '/a.png' })).toBe('image');
		expect(visualKindOf(false, false, { options: [{ image: '/a.png' }] })).toBe('image');
		expect(visualKindOf(false, false, { figures: [{ icon: 'orange' }] })).toBe('icon');
		expect(visualKindOf(false, false, { symbol: 'orange', count: 3 })).toBe('icon');
		expect(visualKindOf(false, false, { options: [{ figures: [{ icon: 'cat' }] }] })).toBe('icon');
		// Falls back to the server flags when the payload carries no stimulus.
		expect(visualKindOf(true, false, { prompt: 'x' })).toBe('icon');
		expect(visualKindOf(false, true, { prompt: 'x' })).toBe('audio');
		expect(visualKindOf(false, false, { prompt: 'x' })).toBe('text');
	});

	it('summarises the pictured objects as Indonesian labels', () => {
		expect(visualSummaryOf({ symbol: 'orange' })).toBe('jeruk');
		expect(visualSummaryOf({ figures: [{ icon: 'orange' }, { icon: 'carrot' }] })).toBe('jeruk, wortel');
		expect(visualSummaryOf({ options: [{ figures: [{ icon: 'orange' }] }, { figures: [{ icon: 'orange' }] }] })).toBe(
			'jeruk'
		);
		expect(visualSummaryOf({ image: '/a.png', image_alt: 'Ilustrasi keluarga.' })).toBe('Ilustrasi keluarga.');
		expect(visualSummaryOf({ prompt: 'x' })).toBe('');
	});

	it('prefers the server has_visual flag', () => {
		expect(hasVisual(detail({ has_visual: true }))).toBe(true);
		expect(hasVisual(detail({ has_visual: false }))).toBe(false);
	});

	it('falls back to content.figures / content.image / content.symbol', () => {
		expect(hasVisual(detail({ payload: { content: { figures: [{ id: 'f1' }] } } }))).toBe(true);
		expect(hasVisual(detail({ payload: { content: { image: '/a.png' } } }))).toBe(true);
		expect(hasVisual(detail({ payload: { content: { symbol: 'orange', count: 3 } } }))).toBe(true);
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

	it('flags early-band questions that still need a visual', () => {
		const early = (over: Record<string, unknown>) =>
			detail({ has_visual: false, has_audio: false, mapping: { phase_code: 'fondasi' }, ...over });
		expect(needsVisual(early({ payload: { type: 'single_choice', content: { prompt: 'x' } } }))).toBe(true);
		expect(needsVisual(early({ payload: { type: 'tap_count', content: { prompt: 'x' } } }))).toBe(true);
		expect(
			needsVisual(early({ payload: { type: 'single_choice', content: { figures: [{ icon: 'orange' }] } } }))
		).toBe(false);
		expect(
			needsVisual(
				detail({
					has_visual: false,
					has_audio: false,
					mapping: { phase_code: 'c' },
					payload: { type: 'short_answer', content: { prompt: 'x' } }
				})
			)
		).toBe(false);
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
