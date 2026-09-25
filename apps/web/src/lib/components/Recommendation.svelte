<script lang="ts">
	// One recommendation; skill-based ones can become a mission for the child.
	import { post, ApiError } from '$lib/api';
	import Flag from 'phosphor-svelte/lib/Flag';
	import Lightbulb from 'phosphor-svelte/lib/Lightbulb';
	import CheckCircle from 'phosphor-svelte/lib/CheckCircle';
	let { rec, childId, missionSkills = [], onmission }: { rec: any; childId: string; missionSkills?: number[]; onmission?: () => void } = $props();
	let status = $state<'idle' | 'busy' | 'done'>('idle');
	let error = $state('');
	const assigned = $derived(status === 'done' || (rec.skill_id && missionSkills.includes(rec.skill_id)));
	const canAssign = $derived(!!rec.skill_id && ['practice', 'remediate', 'continue', 'explore'].includes(rec.kind));
	async function assign() {
		status = 'busy';
		error = '';
		try {
			await post(`/children/${childId}/missions`, { skill_id: rec.skill_id, target: rec.kind === 'explore' ? 2 : 5 });
			status = 'done';
			onmission?.();
		} catch (e) {
			error = e instanceof ApiError ? e.message : 'Gagal.';
			status = 'idle';
		}
	}
</script>

<div class="flex flex-col gap-3 rounded-2xl bg-sun-tint p-4 sm:flex-row sm:items-center">
	<span class="grid size-10 shrink-0 place-items-center rounded-xl bg-white text-tangerine-deep"><Lightbulb size={22} weight="duotone" aria-hidden="true" /></span>
	<div class="min-w-0 flex-1">
		<p class="font-bold">{rec.title}</p>
		<p class="mt-0.5 text-sm text-ink-soft">{rec.detail}</p>
		{#if error}<p class="field-error">{error}</p>{/if}
	</div>
	{#if canAssign}
		{#if assigned}
			<span class="pill anim-pop shrink-0 self-start bg-leaf-tint text-leaf-deep sm:self-center"><CheckCircle size={16} weight="fill" aria-hidden="true" />Jadi misi</span>
		{:else}
			<button class="btn btn-sm shrink-0 self-start bg-white text-ink hover:bg-paper-deep sm:self-center" disabled={status === 'busy'} onclick={assign}>
				<Flag size={18} weight="fill" class="text-tangerine-deep" aria-hidden="true" />Jadikan misi
			</button>
		{/if}
	{/if}
</div>
