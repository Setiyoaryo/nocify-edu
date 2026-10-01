// Creator attribution types (M8).
// Mirrors apps/api/internal/creators and the 0007_creator_attribution migration.

export type CreatorKind = 'system' | 'staff' | 'external';
export type CreatorStatus = 'active' | 'inactive';
export type QualityBand = 'excellent' | 'good' | 'fair' | 'weak' | 'none';

export interface Creator {
	id: string;
	code: string;
	display_name: string;
	kind: CreatorKind;
	user_id: string | null;
	share_pct: number;
	status: CreatorStatus;
	created_at: string;
}

export interface CreatorSummary extends Creator {
	questions: number;
	published: number;
	usage_count: number;
	completion_count: number;
	avg_quality: number;
}

export interface AttributionRow {
	question_id: string;
	code: string;
	type: string;
	status: string;
	skill_id: number;
	skill_name: string;
	usage_count: number;
	completion_count: number;
	correct_count: number;
	first_try_count: number;
	hint_count: number;
	quality_score: number;
	published_at: string | null;
	computed_at: string | null;
}

export interface CreatorDetail {
	creator: CreatorSummary;
	questions: AttributionRow[];
}

const kindLabels: Record<CreatorKind, string> = {
	system: 'Sistem',
	staff: 'Internal',
	external: 'Eksternal'
};

const statusLabels: Record<CreatorStatus, string> = {
	active: 'Aktif',
	inactive: 'Nonaktif'
};

const bandLabels: Record<QualityBand, string> = {
	excellent: 'Sangat baik',
	good: 'Baik',
	fair: 'Cukup',
	weak: 'Perlu perbaikan',
	none: 'Belum ada data'
};

export function kindLabel(kind: string): string {
	return kindLabels[kind as CreatorKind] ?? kind;
}

export function creatorStatusLabel(status: string): string {
	return statusLabels[status as CreatorStatus] ?? status;
}

// qualityBand buckets a 0–100 score. Zero (or missing) means the question has no
// assessment evidence yet, which is not the same as "bad".
export function qualityBand(score: number): QualityBand {
	if (!score) return 'none';
	if (score >= 85) return 'excellent';
	if (score >= 70) return 'good';
	if (score >= 50) return 'fair';
	return 'weak';
}

export function qualityLabel(score: number): string {
	return bandLabels[qualityBand(score)];
}

// shareLabel renders the revenue-share percentage (M8 stores it; no payout math).
export function shareLabel(pct: number): string {
	return `${pct.toLocaleString('id-ID', { maximumFractionDigits: 2 })}%`;
}

// completionRate is answered/served as a whole percentage.
export function completionRate(usage: number, completion: number): number {
	if (!usage) return 0;
	return Math.round((completion / usage) * 100);
}
