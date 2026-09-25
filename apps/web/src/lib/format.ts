const rtf = new Intl.RelativeTimeFormat('id-ID', { numeric: 'auto' });

export function relative(iso: string | null | undefined): string {
	if (!iso) return 'Belum pernah';
	const diff = (new Date(iso).getTime() - Date.now()) / 1000;
	const abs = Math.abs(diff);
	if (abs < 60) return 'Baru saja';
	if (abs < 3600) return rtf.format(Math.round(diff / 60), 'minute');
	if (abs < 86400) return rtf.format(Math.round(diff / 3600), 'hour');
	return rtf.format(Math.round(diff / 86400), 'day');
}

export const date = (iso: string) => new Date(iso).toLocaleDateString('id-ID', { day: 'numeric', month: 'long', year: 'numeric' });
export const dateTime = (iso: string) =>
	new Date(iso).toLocaleString('id-ID', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' });
export const rupiah = (n: number) => 'Rp' + n.toLocaleString('id-ID');
