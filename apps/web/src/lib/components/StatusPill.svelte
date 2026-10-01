<script lang="ts">
	// Workflow status as colour + word (never colour alone).
	let { status }: { status: string } = $props();
	const map: Record<string, [string, string]> = {
		draft: ['Draft', 'bg-paper-deep text-ink-soft'],
		review: ['Review', 'bg-sky-tint text-[#1a6fa3]'],
		approved: ['Disetujui', 'bg-primary-tint text-primary-deep'],
		published: ['Tayang', 'bg-leaf-tint text-leaf-deep'],
		retired: ['Arsip', 'bg-paper-deep text-ink-soft'],
		imported: ['Diimpor', 'bg-paper-deep text-ink-soft'],
		license_verified: ['Lisensi OK', 'bg-sky-tint text-[#1a6fa3]'],
		design_approved: ['Desain OK', 'bg-primary-tint text-primary-deep'],
		performance_checked: ['Performa OK', 'bg-sun-tint text-ink'],
		rejected: ['Ditolak', 'bg-rose-tint text-[#a8264f]'],
		// Pipeline candidate + publication statuses (M5).
		needs_review: ['Perlu tinjauan', 'bg-sun-tint text-ink'],
		queued: ['Menunggu tayang', 'bg-sky-tint text-[#1a6fa3]'],
		publishing: ['Menayangkan', 'bg-sky-tint text-[#1a6fa3]'],
		not_applicable: ['Tidak diterbitkan', 'bg-paper-deep text-ink-soft'],
		changes_requested: ['Minta perbaikan', 'bg-sun-tint text-ink'],
		failed: ['Gagal tayang', 'bg-rose-tint text-[#a8264f]'],
		// Curriculum change review + event statuses (M7).
		pending: ['Belum ditinjau', 'bg-sun-tint text-ink'],
		acknowledged: ['Diterima', 'bg-primary-tint text-primary-deep'],
		dismissed: ['Diabaikan', 'bg-paper-deep text-ink-soft'],
		deferred: ['Ditunda', 'bg-sky-tint text-[#1a6fa3]'],
		detected: ['Terdeteksi', 'bg-sun-tint text-ink'],
		analyzed: ['Dianalisis', 'bg-primary-tint text-primary-deep'],
		superseded: ['Digantikan', 'bg-paper-deep text-ink-soft'],
		// Creator statuses (M8).
		active: ['Aktif', 'bg-leaf-tint text-leaf-deep'],
		inactive: ['Nonaktif', 'bg-paper-deep text-ink-soft']
	};
	const m = $derived(map[status] ?? [status, 'bg-paper-deep text-ink-soft']);
</script>

<span class="pill {m[1]}"><span class="size-1.5 rounded-full bg-current" aria-hidden="true"></span>{m[0]}</span>
