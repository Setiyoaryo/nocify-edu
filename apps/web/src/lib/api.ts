// Thin JSON client for the Go API. Same-origin: in development Vite proxies
// /api, in production Traefik does. Session is an HttpOnly cookie.

export class ApiError extends Error {
	constructor(
		public status: number,
		public code: string,
		message: string,
		public fields: Record<string, string> = {}
	) {
		super(message);
	}
}

type Method = 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE';

export async function api<T = any>(method: Method, path: string, body?: unknown): Promise<T> {
	const init: RequestInit = { method, credentials: 'same-origin', headers: {} };
	if (body instanceof FormData) {
		init.body = body;
	} else if (body !== undefined) {
		(init.headers as Record<string, string>)['Content-Type'] = 'application/json';
		init.body = JSON.stringify(body);
	}
	let res: Response;
	try {
		res = await fetch('/api/v1' + path, init);
	} catch {
		throw new ApiError(0, 'offline', 'Tidak dapat terhubung. Periksa koneksi internet lalu coba lagi.');
	}
	if (res.status === 204) return undefined as T;
	const data = await res.json().catch(() => ({}));
	if (!res.ok) {
		throw new ApiError(res.status, data.error ?? 'error', data.message ?? 'Terjadi kesalahan.', data.fields ?? {});
	}
	return data as T;
}

export const get = <T = any>(p: string) => api<T>('GET', p);
export const post = <T = any>(p: string, b?: unknown) => api<T>('POST', p, b ?? {});
export const patch = <T = any>(p: string, b: unknown) => api<T>('PATCH', p, b);
export const put = <T = any>(p: string, b: unknown) => api<T>('PUT', p, b);
export const del = <T = any>(p: string, b?: unknown) => api<T>('DELETE', p, b);

/** Fire-and-forget first-party analytics for allow-listed client events (§50). */
export function track(name: 'landing_visit' | 'pricing_viewed' | 'progress_viewed' | 'subjects_viewed', props: Record<string, string> = {}) {
	fetch('/api/v1/events', {
		method: 'POST',
		credentials: 'same-origin',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ name, props }),
		keepalive: true
	}).catch(() => {});
}
