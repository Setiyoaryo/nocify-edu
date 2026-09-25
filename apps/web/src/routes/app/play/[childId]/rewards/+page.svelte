<script lang="ts">
	import { get } from '$lib/api';
	import { page } from '$app/state';
	import Figure from '$lib/components/Figure.svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import Star from 'phosphor-svelte/lib/Star';
	import Lightning from 'phosphor-svelte/lib/Lightning';
	import LockSimple from 'phosphor-svelte/lib/LockSimple';
	let data = $state<any>(null);
	get(`/children/${page.params.childId}/rewards`).then((r) => (data = r));
	const sections = [
		{ kind: 'badge', title: 'Lencana', tone: 'pastel-sun' },
		{ kind: 'sticker', title: 'Stiker', tone: 'pastel-rose' },
		{ kind: 'avatar', title: 'Avatar', tone: 'pastel-grape' },
		{ kind: 'scene', title: 'Tempat', tone: 'pastel-sky' }
	];
</script>

<svelte:head><title>Hadiah | EduPlay</title></svelte:head>

<h1 class="anim-rise mt-3 text-3xl sm:text-4xl">Hadiahku</h1>
{#if data}
	<div class="mt-4 grid grid-cols-2 gap-3 sm:max-w-md">
		<p class="pastel-grape flex items-center gap-3 rounded-[1.25rem] p-4"><span class="grid size-11 place-items-center rounded-full bg-white text-primary"><Lightning size={24} weight="fill" aria-hidden="true" /></span><span><b class="block text-xl">{data.child.xp}</b><span class="text-sm text-ink-soft">XP</span></span></p>
		<p class="pastel-sun flex items-center gap-3 rounded-[1.25rem] p-4"><span class="grid size-11 place-items-center rounded-full bg-white text-sun-deep"><Star size={24} weight="fill" aria-hidden="true" /></span><span><b class="block text-xl">{data.stars}</b><span class="text-sm text-ink-soft">bintang</span></span></p>
	</div>
	{#each sections as sec}
		<section class="mt-8">
			<h2 class="text-xl">{sec.title}</h2>
			<ul class="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
				{#each data.items.filter((i: any) => i.kind === sec.kind) as it, n (it.code)}
					<li style="--d:{n * 60}ms" class="anim-rise relative flex flex-col items-center rounded-[1.25rem] p-4 text-center {it.owned ? sec.tone : 'bg-paper-deep/70'}">
						<span class={it.owned ? '' : 'opacity-35 grayscale'}>
							{#if it.kind === 'avatar'}<Avatar code={it.icon} size={64} />{:else}<span class="grid size-16 place-items-center"><Figure name={it.icon} size={52} /></span>{/if}
						</span>
						<span class="mt-2 font-extrabold">{it.name}</span>
						<span class="mt-0.5 text-xs font-semibold text-ink-soft">
							{#if it.owned}Punyaku{:else if it.kind === 'badge'}{it.description}{:else}{it.xp_required} XP{/if}
						</span>
						{#if !it.owned}<LockSimple size={18} weight="fill" class="absolute top-3 right-3 text-ink-soft" aria-label="Terkunci" />{/if}
					</li>
				{/each}
			</ul>
		</section>
	{/each}
{/if}
