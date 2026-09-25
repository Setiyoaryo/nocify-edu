<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	// Cheerful card (Ruangguru-style): pastel wash, white icon disc, faint big
	// icon as decoration, optional white pill action. Renders as <a>, <button>
	// or <div> depending on href/onclick.
	type Tone = 'grape' | 'rose' | 'peach' | 'sun' | 'mint' | 'sky' | 'lilac';
	let {
		tone = 'grape',
		icon,
		iconColor = '#5b5ce2',
		title,
		desc = '',
		cta = '',
		href,
		onclick,
		disabled = false,
		size = 'md',
		children,
		class: klass = ''
	}: {
		tone?: Tone;
		icon: Component<any>;
		iconColor?: string;
		title: string;
		desc?: string;
		cta?: string;
		href?: string;
		onclick?: () => void;
		disabled?: boolean;
		size?: 'md' | 'lg';
		children?: Snippet;
		class?: string;
	} = $props();
	const Icon = $derived(icon);
	const tag = $derived(href ? 'a' : onclick ? 'button' : 'div');
</script>

<svelte:element
	this={tag}
	{href}
	{onclick}
	role={tag === 'button' ? 'button' : tag === 'a' ? 'link' : undefined}
	disabled={tag === 'button' ? disabled : undefined}
	type={tag === 'button' ? 'button' : undefined}
	class="fun-card pastel-{tone} relative flex h-full w-full flex-col overflow-hidden rounded-[1.4rem] text-left text-ink {size === 'lg' ? 'p-6 sm:p-7' : 'p-5'} {tag !== 'div' ? 'press lift' : ''} disabled:opacity-60 {klass}"
>
	<span class="deco pointer-events-none absolute -right-6 -bottom-8 opacity-[0.13]" aria-hidden="true"><Icon size={size === 'lg' ? 170 : 130} weight="fill" color={iconColor} /></span>
	<span class="relative grid shrink-0 place-items-center rounded-full bg-white shadow-sm {size === 'lg' ? 'size-14' : 'size-12'}" aria-hidden="true">
		<Icon size={size === 'lg' ? 30 : 26} weight="duotone" color={iconColor} />
	</span>
	<span class="relative mt-4 block font-extrabold leading-tight {size === 'lg' ? 'text-2xl' : 'text-lg'}">{title}</span>
	{#if desc}<span class="relative mt-1.5 block text-sm font-semibold text-ink-soft">{desc}</span>{/if}
	{#if children}<span class="relative mt-3 block">{@render children()}</span>{/if}
	{#if cta}<span class="relative mt-auto block pt-5"><span class="pill-action w-full">{cta}</span></span>{/if}
</svelte:element>

<style>
	.deco {
		transition: transform 0.5s var(--ease-spring);
		transform: rotate(-12deg);
	}
	.fun-card:hover .deco,
	.fun-card:active .deco {
		transform: rotate(-4deg) scale(1.06);
	}
</style>
