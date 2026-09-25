<script lang="ts">
	import { get } from '$lib/api';
	import { allAvatars, starterAvatars } from '$lib/avatars';
	import { figure } from '$lib/figures';
	import Avatar from '$lib/components/Avatar.svelte';
	import LockSimple from 'phosphor-svelte/lib/LockSimple';
	import type { Grade } from '$lib/types';

	let {
		value = $bindable(),
		fields = {},
		unlockedAvatars = starterAvatars,
		editing = false
	}: {
		value: { nickname: string; grade_id: number | null; avatar: string; pin: string };
		fields?: Record<string, string>;
		unlockedAvatars?: string[];
		editing?: boolean;
	} = $props();

	let grades = $state<Grade[]>([]);
	get('/grades').then((r) => (grades = r.grades));
</script>

<div class="space-y-6">
	<div>
		<label class="label" for="nickname">Nama panggilan</label>
		<input class="field" id="nickname" maxlength="30" autocomplete="off" bind:value={value.nickname} aria-describedby="nick-hint" aria-invalid={!!fields.nickname} />
		<p id="nick-hint" class={fields.nickname ? 'field-error' : 'mt-1 text-sm text-ink-soft'}>
			{fields.nickname ?? 'Nama panggilan saja, tidak perlu nama lengkap.'}
		</p>
	</div>
	<div>
		<label class="label" for="grade">Kelas</label>
		<select class="field" id="grade" bind:value={value.grade_id} aria-invalid={!!fields.grade_id}>
			<option value={null} disabled>Pilih kelas</option>
			{#each grades as g}<option value={g.id}>{g.level_name} – {g.name}</option>{/each}
		</select>
		{#if fields.grade_id}<p class="field-error">{fields.grade_id}</p>{/if}
	</div>
	<fieldset>
		<legend class="label">Avatar</legend>
		<div class="flex flex-wrap gap-3">
			{#each allAvatars as a}
				{@const locked = !unlockedAvatars.includes(a)}
				<label class="relative cursor-pointer rounded-full p-1 ring-2 transition has-[:disabled]:cursor-not-allowed has-[:disabled]:opacity-45 has-[:focus-visible]:outline-3 has-[:focus-visible]:outline-primary {value.avatar === a ? 'ring-primary' : 'ring-transparent'}"
					title={locked ? 'Terbuka dengan XP' : figure(a).label}>
					<input type="radio" class="sr-only" name="avatar" value={a} bind:group={value.avatar} disabled={locked} aria-label={locked ? `${figure(a).label} (terkunci)` : figure(a).label} />
					<Avatar code={a} size={56} />
					{#if locked}<span class="absolute right-0 bottom-0 grid size-6 place-items-center rounded-full bg-ink text-white"><LockSimple size={14} weight="fill" aria-hidden="true" /></span>{/if}
				</label>
			{/each}
		</div>
	</fieldset>
	<div>
		<label class="label" for="pin">PIN anak (opsional)</label>
		<input class="field max-w-40 tracking-[0.5em]" id="pin" inputmode="numeric" maxlength="4" pattern="[0-9]*" autocomplete="off" bind:value={value.pin} aria-describedby="pin-hint" />
		<p id="pin-hint" class={fields.pin ? 'field-error' : 'mt-1 text-sm text-ink-soft'}>
			{fields.pin ?? (editing ? '4 angka. Kosongkan untuk tidak mengubah PIN.' : '4 angka, supaya kakak-adik tidak saling tertukar profil.')}
		</p>
	</div>
</div>
