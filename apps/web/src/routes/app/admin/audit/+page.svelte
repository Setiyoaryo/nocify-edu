<script lang="ts">
	import { get } from '$lib/api';
	import { dateTime } from '$lib/format';
	let logs = $state<any[]>([]);
	get('/admin/audit-logs').then((r) => (logs = r.logs));
</script>

<svelte:head><title>Audit | CMS</title></svelte:head>
<h1 class="text-3xl sm:text-4xl">Log audit</h1>
<p class="mt-1 text-ink-soft">200 perubahan terakhir oleh staf dan sistem.</p>
<div class="card mt-6 overflow-x-auto">
	<table class="w-full min-w-2xl text-left text-sm">
		<thead class="bg-paper-deep/60 text-xs font-bold text-ink-soft"><tr><th class="px-4 py-3">Waktu</th><th>Pelaku</th><th>Aksi</th><th>Entitas</th><th>Data</th></tr></thead>
		<tbody>
			{#each logs as l (l.id)}
				<tr class="border-t border-line align-top [&>td]:px-4">
					<td class="py-3 whitespace-nowrap">{dateTime(l.created_at)}</td><td>{l.actor}</td><td>{l.action}</td>
					<td>{l.entity_type}<br /><span class="text-xs text-ink-soft">{l.entity_id}</span></td>
					<td class="max-w-xs text-xs break-all text-ink-soft">{JSON.stringify(l.data)}</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>
