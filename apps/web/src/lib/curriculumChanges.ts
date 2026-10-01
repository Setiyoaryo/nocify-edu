// Reviewer CMS types for detected curriculum changes (M7).
// Mirrors apps/api/internal/changes and services/content-pipeline/sql/0007_curriculum_changes.sql.

export type ReviewStatus = 'pending' | 'acknowledged' | 'dismissed' | 'deferred';
export type EventStatus = 'detected' | 'analyzed' | 'failed' | 'superseded';
export type ChangeType = 'added' | 'changed' | 'removed';
export type MatchKind = 'skill' | 'provenance';

export interface ChangeSummary {
	event_id: number;
	document_code: string;
	url: string;
	curriculum_version_id: number | null;
	baseline_hash: string;
	observed_hash: string;
	detection: string;
	document_type: string;
	cp_added: number;
	cp_changed: number;
	cp_removed: number;
	event_status: EventStatus;
	error: string;
	impact_count: number;
	affected_questions: number;
	review_status: ReviewStatus;
	reviewer: string | null;
	decided_at: string | null;
	created_at: string;
}

export interface Review {
	id: number;
	event_id: number;
	status: ReviewStatus;
	notes: string;
	reviewer: string | null;
	decided_at: string | null;
}

export interface ImpactQuestion {
	question_id: string;
	question_code: string;
	skill_id: number;
	skill_name: string;
	match: MatchKind;
	status: string;
}

export interface Impact {
	id: number;
	cp_id: number | null;
	cp_code: string;
	change_type: ChangeType;
	old_statement: string;
	new_statement: string;
	statement_hash: string;
	affected_skills: number;
	affected_questions: number;
	questions: ImpactQuestion[];
}

export interface ChangeDetail extends ChangeSummary {
	impacts: Impact[];
	review: Review | null;
}

const reviewLabels: Record<ReviewStatus, string> = {
	pending: 'Belum ditinjau',
	acknowledged: 'Diterima',
	dismissed: 'Diabaikan',
	deferred: 'Ditunda'
};

const eventLabels: Record<string, string> = {
	detected: 'Terdeteksi',
	analyzed: 'Dianalisis',
	failed: 'Gagal dianalisis',
	superseded: 'Digantikan'
};

const changeTypeLabels: Record<ChangeType, string> = {
	added: 'Ditambahkan',
	changed: 'Diubah',
	removed: 'Dihapus'
};

export function reviewStatusLabel(status: string): string {
	return reviewLabels[status as ReviewStatus] ?? status;
}

export function eventStatusLabel(status: string): string {
	return eventLabels[status] ?? status;
}

export function changeTypeLabel(type: string): string {
	return changeTypeLabels[type as ChangeType] ?? type;
}

export function matchLabel(match: string): string {
	return match === 'provenance' ? 'Provenance' : 'Skill';
}

// Short hash for display; the full value stays available in the tooltip.
export function shortHash(hash: string, length = 10): string {
	if (!hash) return '—';
	return hash.length > length ? hash.slice(0, length) : hash;
}
