// Content Studio: browse pipeline content per lesson (skill) — grade → subject
// → chapter → skill — and see each material with the questions derived from it
// (lineage = same chunk), plus the canonical questions already published.
// Mirrors apps/api/internal/candidates/content_studio.go.

import type { CandidateDetail } from './candidates';
import { bandFor } from './candidates';
import { figures } from './figures';

export interface StudioSkill {
	id: number;
	code: string;
	name: string;
	materials: number;
	questions: number;
	needs_review: number;
	published: number;
}

export interface StudioChapter {
	id: number;
	code: string;
	name: string;
	sort: number;
	skills: StudioSkill[];
}

export interface StudioSubject {
	id: number;
	code: string;
	name: string;
	sort: number;
	chapters: StudioChapter[];
}

export interface StudioGrade {
	id: number;
	code: string;
	name: string;
	sort: number;
	subjects: StudioSubject[];
}

export interface StudioTree {
	available: boolean;
	unmapped: { materials: number; questions: number };
	grades: StudioGrade[];
}

export interface StudioLesson {
	id: number;
	code: string;
	name: string;
	description: string;
	chapter_id: number;
	chapter_name: string;
	grade_id: number;
	grade_code: string;
	grade_name: string;
	subject_id: number;
	subject_code: string;
	subject_name: string;
}

export interface StudioMaterialGroup {
	material: CandidateDetail;
	questions: CandidateDetail[];
}

export interface CanonicalQuestion {
	id: string;
	code: string;
	type: string;
	difficulty: number | null;
	content: Record<string, unknown>;
	answer: unknown;
	explanation: string;
	hint: string;
	published_at: string | null;
	reviewer: string;
	has_visual: boolean;
	has_audio: boolean;
}

export interface LessonDetail {
	available: boolean;
	skill: StudioLesson;
	materials: StudioMaterialGroup[];
	orphan_questions: CandidateDetail[];
	published: CanonicalQuestion[];
}

export type VisualKind = 'image' | 'icon' | 'audio' | 'text';

const VISUAL_KIND_LABELS: Record<VisualKind, string> = {
	image: 'Visual: gambar',
	icon: 'Visual: ikon',
	audio: 'Butuh audio',
	text: 'Teks saja'
};

export function visualKindLabel(kind: VisualKind): string {
	return VISUAL_KIND_LABELS[kind];
}

function hasImage(content: Record<string, any>): boolean {
	if (content.image) return true;
	return Array.isArray(content.options) && content.options.some((o: any) => !!o?.image);
}

function hasIcon(content: Record<string, any>): boolean {
	if (Array.isArray(content.figures) && content.figures.length) return true;
	if (content.symbol) return true;
	return (
		Array.isArray(content.options) &&
		content.options.some((o: any) => Array.isArray(o?.figures) && o.figures.length)
	);
}

/**
 * Whether a question renders a picture. Prefers the server flag; falls back to
 * inspecting the payload (content.figures / content.image / content.symbol, or
 * per-option figures/image for image_choice).
 */
export function hasVisual(detail: Pick<CandidateDetail, 'has_visual' | 'payload'>): boolean {
	if (typeof detail.has_visual === 'boolean') return detail.has_visual;
	const content = detail.payload?.content as Record<string, any> | undefined;
	if (!content) return false;
	return hasIcon(content) || hasImage(content);
}

export function hasAudio(detail: Pick<CandidateDetail, 'has_audio' | 'payload'>): boolean {
	if (typeof detail.has_audio === 'boolean') return detail.has_audio;
	const content = detail.payload?.content as Record<string, any> | undefined;
	return !!content?.audio_text;
}

/** The stimulus a question shows: an illustration, icons, audio only, or text. */
export function visualKindOf(
	hasVisualFlag: boolean,
	hasAudioFlag: boolean,
	content: Record<string, any> | undefined
): VisualKind {
	const c = content ?? {};
	if (hasImage(c)) return 'image';
	if (hasIcon(c)) return 'icon';
	if (hasVisualFlag) return 'icon';
	if (hasAudioFlag) return 'audio';
	return 'text';
}

/** The pictured objects of a question as Indonesian labels (deduped). */
export function visualSummaryOf(content: Record<string, any> | undefined): string {
	const c = content ?? {};
	if (c.image) return String(c.image_alt ?? '');
	const names: string[] = [];
	if (c.symbol) names.push(String(c.symbol));
	for (const f of c.figures ?? []) if (f?.icon) names.push(String(f.icon));
	for (const o of c.options ?? []) for (const f of o?.figures ?? []) if (f?.icon) names.push(String(f.icon));
	const seen = new Set<string>();
	const out: string[] = [];
	for (const name of names) {
		const label = figures[name]?.label ?? name;
		if (!seen.has(label)) {
			seen.add(label);
			out.push(label);
		}
	}
	return out.join(', ');
}

/** True when an early-band question still has no stimulus — the review gap to fix. */
export function needsVisual(detail: CandidateDetail): boolean {
	const content = detail.payload?.content as Record<string, any> | undefined;
	if (visualKindOf(hasVisual(detail), hasAudio(detail), content) !== 'text') return false;
	const type = detail.payload?.type ?? '';
	if (type === 'image_choice' || type === 'tap_count') return true;
	return bandFor(detail.mapping?.phase_code ?? '') === 'early';
}

/** A skill has something to show if it has any candidate or published question. */
export function skillHasWork(s: StudioSkill): boolean {
	return s.materials + s.questions + s.published > 0;
}
