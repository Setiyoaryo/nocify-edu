// Reviewer CMS types for content-pipeline candidates (M5).
// Mirrors apps/api/internal/candidates and packages/contracts/*-candidate.schema.json.

export type CandidateKind = 'material' | 'question';

export interface QaIssue {
	code: string;
	message: string;
	blocking: boolean;
}

export interface CandidateSummary {
	kind: CandidateKind;
	id: number;
	source_id: number;
	source_code: string;
	label: string;
	status: string;
	confidence: number;
	blocking_issues: number;
	skill_id: number | null;
	skill_name: string;
	phase_code: string;
	grade_code: string;
	subject_code: string;
	publication_status: string | null;
	decision: string | null;
	updated_at: string;
}

export interface Publication {
	id: number;
	kind: CandidateKind;
	candidate_id: number;
	decision: string;
	status: string;
	notes: string;
	checklist: Record<string, boolean>;
	reviewer: string | null;
	decided_at: string;
	canonical_question_id: string | null;
	attempts: number;
	last_error: string;
	published_at: string | null;
}

export interface SourceRef {
	id: number;
	code: string;
	name: string;
	base_url: string;
}

export interface Lineage {
	document_id: number;
	document_title: string;
	canonical_url: string;
	retrieved_at: string | null;
	license: string;
	chunk_id: number;
	heading: string;
	text: string;
	page_start: number | null;
	page_end: number | null;
}

export interface Mapping {
	skill_id: number | null;
	skill_name: string;
	skill_code: string;
	cp_id: number | null;
	cp_code: string;
	cp_statement: string;
	phase_code: string;
	phase_name: string;
	grade_code: string;
	grade_name: string;
	subject_code: string;
	subject_name: string;
}

export interface CandidateDetail {
	kind: CandidateKind;
	id: number;
	source_id: number;
	status: string;
	confidence: number;
	qa_issues: QaIssue[];
	ai_generated: boolean;
	content_hash: string;
	language: string;
	template: string;
	type: string;
	difficulty: number | null;
	learning_objective: string;
	title: string;
	summary: string;
	prompt: string;
	payload?: { type: string; content: Record<string, unknown>; answer: unknown; explanation: string } | null;
	readability?: Record<string, unknown> | null;
	source: SourceRef;
	lineage: Lineage;
	mapping: Mapping;
	publication: Publication | null;
	created_at: string;
	updated_at: string;
}

// Preview band for a generated question, derived from the curriculum phase (§30).
export function bandFor(phaseCode: string): 'early' | 'lower' | 'upper' {
	if (phaseCode === 'fondasi' || phaseCode === 'a') return 'early';
	if (phaseCode === 'b') return 'lower';
	return 'upper';
}

export function kindLabel(kind: string): string {
	return kind === 'question' ? 'Soal' : 'Materi';
}

export function qaIssues(raw: unknown): QaIssue[] {
	return Array.isArray(raw) ? (raw as QaIssue[]) : [];
}

export function blockingCount(issues: QaIssue[]): number {
	return issues.filter((i) => i.blocking).length;
}

export function confidenceLabel(confidence: number): string {
	return `${Math.round(confidence * 100)}%`;
}
