export type Role = 'parent' | 'content_editor' | 'reviewer' | 'admin';

export interface User {
	id: string;
	email: string;
	display_name: string;
	role: Role;
}

export interface Child {
	id: string;
	nickname: string;
	grade_id: number;
	grade_name: string;
	ux_band: 'early' | 'lower' | 'upper' | 'secondary';
	avatar: string;
	has_pin: boolean;
	xp: number;
	level: number;
	last_activity: string | null;
}

export interface Grade {
	id: number;
	code: string;
	name: string;
	level: string;
	level_name: string;
	session_size: number;
	ux_band: string;
}

export interface Fig {
	icon: string;
	count?: number;
	color?: string;
}
export interface Option {
	id: string;
	label: string;
	figures?: Fig[];
	image?: string;
	alt?: string;
}
export interface Item {
	id: string;
	label: string;
	icon?: string;
	color?: string;
}

/** Child-facing question content (answers are never included). */
export interface Content {
	prompt: string;
	audio_text?: string;
	figures?: Fig[];
	image?: string;
	image_alt?: string;
	options?: Option[];
	symbol?: string;
	color?: string;
	count?: number;
	choices?: number[];
	unit?: string;
	items?: Item[];
	left?: Item[];
	right?: Item[];
}

export type Response = {
	option_id?: string;
	value?: number;
	text?: string;
	order?: string[];
	pairs?: Record<string, string>;
};

export interface Entitlement {
	plan: 'free' | 'monthly';
	premium: boolean;
	status?: string;
	current_period_end?: string;
	cancel_at_period_end: boolean;
	limits: { max_children: number; daily_challenges: number };
}

export interface RecapStats {
	total_challenges: number;
	total_questions: number;
	accuracy: number;
	stars: number;
	mastered_skills: number;
	streak: number;
	favorite_subject: string;
	top_badge?: {
		name: string;
		icon: string;
		description: string;
	};
}

export interface Archetype {
	code: string;
	title: string;
	subtitle: string;
	praise: string;
	highlight: string;
	tag: string;
	theme: string;
	icon: string;
	tier: 'starter' | 'rare' | 'epic' | 'legendary';
	tier_name: string;
	next_tier_hint: string;
}

export interface ChildRecap {
	child: Child;
	stats: RecapStats;
	archetype: Archetype;
}
