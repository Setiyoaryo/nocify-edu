// Content Studio: browse pipeline content per lesson (skill) — grade → subject
// → chapter → skill — and see each material with the questions derived from it
// (lineage = same chunk), plus the canonical questions already published.
// Mirrors apps/api/internal/candidates/content_studio.go.

import type { CandidateDetail } from './candidates';

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

/** Short label for a question's visual treatment. */
export function visualLabel(hasVisual: boolean, hasAudio: boolean): string {
	if (hasVisual) return 'Ada visual';
	if (hasAudio) return 'Butuh audio';
	return 'Teks saja';
}

/**
 * Whether a question renders a picture. Prefers the server flag; falls back to
 * inspecting the payload (content.figures / content.image, or per-option
 * figures/image for image_choice).
 */
export function hasVisual(detail: Pick<CandidateDetail, 'has_visual' | 'payload'>): boolean {
	if (typeof detail.has_visual === 'boolean') return detail.has_visual;
	const content = detail.payload?.content as Record<string, any> | undefined;
	if (!content) return false;
	if (Array.isArray(content.figures) && content.figures.length) return true;
	if (content.image) return true;
	if (Array.isArray(content.options)) {
		return content.options.some(
			(o: any) => (Array.isArray(o?.figures) && o.figures.length) || o?.image
		);
	}
	return false;
}

export function hasAudio(detail: Pick<CandidateDetail, 'has_audio' | 'payload'>): boolean {
	if (typeof detail.has_audio === 'boolean') return detail.has_audio;
	const content = detail.payload?.content as Record<string, any> | undefined;
	return !!content?.audio_text;
}

/** A skill has something to show if it has any candidate or published question. */
export function skillHasWork(s: StudioSkill): boolean {
	return s.materials + s.questions + s.published > 0;
}
