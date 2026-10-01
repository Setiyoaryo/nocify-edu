import { describe, expect, it } from 'vitest';
import {
	changeTypeLabel,
	eventStatusLabel,
	matchLabel,
	reviewStatusLabel,
	shortHash
} from './curriculumChanges';

describe('curriculumChanges', () => {
	it('labels review statuses', () => {
		expect(reviewStatusLabel('pending')).toBe('Belum ditinjau');
		expect(reviewStatusLabel('acknowledged')).toBe('Diterima');
		expect(reviewStatusLabel('dismissed')).toBe('Diabaikan');
		expect(reviewStatusLabel('deferred')).toBe('Ditunda');
		expect(reviewStatusLabel('other')).toBe('other');
	});

	it('labels event statuses', () => {
		expect(eventStatusLabel('analyzed')).toBe('Dianalisis');
		expect(eventStatusLabel('failed')).toBe('Gagal dianalisis');
	});

	it('labels change types and match kinds', () => {
		expect(changeTypeLabel('added')).toBe('Ditambahkan');
		expect(changeTypeLabel('changed')).toBe('Diubah');
		expect(changeTypeLabel('removed')).toBe('Dihapus');
		expect(matchLabel('skill')).toBe('Skill');
		expect(matchLabel('provenance')).toBe('Provenance');
	});

	it('shortens hashes for display', () => {
		expect(shortHash('abcdef0123456789')).toBe('abcdef0123');
		expect(shortHash('short')).toBe('short');
		expect(shortHash('')).toBe('—');
	});
});
