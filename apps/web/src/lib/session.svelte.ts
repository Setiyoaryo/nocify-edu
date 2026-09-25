import { get, post } from './api';
import type { User } from './types';

class Session {
	user = $state<User | null>(null);
	loaded = $state(false);

	async load() {
		try {
			this.user = (await get<{ user: User }>('/me')).user;
		} catch {
			this.user = null;
		}
		this.loaded = true;
		return this.user;
	}

	async logout() {
		await post('/auth/logout');
		this.user = null;
	}

	get isStaff() {
		return !!this.user && this.user.role !== 'parent';
	}
}

export const session = new Session();
