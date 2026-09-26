<script lang="ts">
	import type { Archetype, Child } from '$lib/types';
	import Avatar from './Avatar.svelte';

	let { child, archetype }: { child: Child; archetype: Archetype } = $props();

	// Palette & atmosphere per archetype so each card feels distinct and custom
	const themeData = $derived.by(() => {
		switch (archetype.code) {
			case 'ksatria_api':
				return {
					skyTop: '#ff5e36',
					skyMid: '#ff9900',
					skyBottom: '#ffe8b3',
					accentGlow: 'rgba(255, 94, 54, 0.45)',
					wingColor: '#ffd15c',
					wingShade: '#d97706',
					mascotCrown: '#ffbe1a',
					mascotBody: '#ffffff',
					mascotFace: '#ff4d00',
					heroAccessory: 'api',
					gemColor: '#ef4444'
				};
			case 'detektif_elang':
				return {
					skyTop: '#0f4c81',
					skyMid: '#2563eb',
					skyBottom: '#bae6fd',
					accentGlow: 'rgba(37, 99, 235, 0.45)',
					wingColor: '#60a5fa',
					wingShade: '#1d4ed8',
					mascotCrown: '#38bdf8',
					mascotBody: '#f8fafc',
					mascotFace: '#0284c7',
					heroAccessory: 'elang',
					gemColor: '#0284c7'
				};
			case 'penyihir_angka':
				return {
					skyTop: '#4c1d95',
					skyMid: '#7c3aed',
					skyBottom: '#e9d5ff',
					accentGlow: 'rgba(124, 58, 237, 0.45)',
					wingColor: '#c084fc',
					wingShade: '#6d28d9',
					mascotCrown: '#fbbf24',
					mascotBody: '#faf5ff',
					mascotFace: '#8b5cf6',
					heroAccessory: 'angka',
					gemColor: '#9333ea'
				};
			case 'penjelajah_kata':
				return {
					skyTop: '#064e3b',
					skyMid: '#059669',
					skyBottom: '#a7f3d0',
					accentGlow: 'rgba(5, 150, 105, 0.45)',
					wingColor: '#34d399',
					wingShade: '#047857',
					mascotCrown: '#facc15',
					mascotBody: '#f0fdf4',
					mascotFace: '#10b981',
					heroAccessory: 'kata',
					gemColor: '#10b981'
				};
			case 'kolektor_bintang':
				return {
					skyTop: '#d97706',
					skyMid: '#fbbf24',
					skyBottom: '#fef3c7',
					accentGlow: 'rgba(251, 191, 36, 0.55)',
					wingColor: '#fde047',
					wingShade: '#b45309',
					mascotCrown: '#fef08a',
					mascotBody: '#ffffff',
					mascotFace: '#f59e0b',
					heroAccessory: 'bintang',
					gemColor: '#f59e0b'
				};
			case 'ksatria_kilat':
				return {
					skyTop: '#1e1b4b',
					skyMid: '#4338ca',
					skyBottom: '#c7d2fe',
					accentGlow: 'rgba(67, 56, 202, 0.45)',
					wingColor: '#fbbf24',
					wingShade: '#4338ca',
					mascotCrown: '#facc15',
					mascotBody: '#ffffff',
					mascotFace: '#4f46e5',
					heroAccessory: 'kilat',
					gemColor: '#4f46e5'
				};
			default:
				return {
					skyTop: '#1d4ed8',
					skyMid: '#38bdf8',
					skyBottom: '#fef08a',
					accentGlow: 'rgba(56, 189, 248, 0.45)',
					wingColor: '#fde047',
					wingShade: '#d97706',
					mascotCrown: '#ffbe1a',
					mascotBody: '#ffffff',
					mascotFace: '#2563eb',
					heroAccessory: 'pahlawan',
					gemColor: '#38bdf8'
				};
		}
	});
</script>

<div class="relative w-full overflow-hidden rounded-t-[1.6rem] aspect-[16/14]">
	<!-- Sky and Atmosphere Canvas -->
	<svg viewBox="0 0 500 420" class="absolute inset-0 size-full" preserveAspectRatio="xMidYMid slice" aria-hidden="true">
		<defs>
			<linearGradient id="skyGrad-{archetype.code}" x1="0" y1="0" x2="0" y2="1">
				<stop offset="0%" stop-color={themeData.skyTop} />
				<stop offset="55%" stop-color={themeData.skyMid} />
				<stop offset="100%" stop-color={themeData.skyBottom} />
			</linearGradient>

			<radialGradient id="sunGlow-{archetype.code}" cx="50%" cy="30%" r="60%">
				<stop offset="0%" stop-color="#ffffff" stop-opacity="0.9" />
				<stop offset="40%" stop-color="#fff5cc" stop-opacity="0.5" />
				<stop offset="100%" stop-color={themeData.skyMid} stop-opacity="0" />
			</radialGradient>

			<linearGradient id="goldRibbon" x1="0" y1="0" x2="1" y2="1">
				<stop offset="0%" stop-color="#fff3b0" />
				<stop offset="50%" stop-color="#f59e0b" />
				<stop offset="100%" stop-color="#b45309" />
			</linearGradient>

			<filter id="glowFilter" x="-20%" y="-20%" width="140%" height="140%">
				<feGaussianBlur stdDeviation="6" result="blur" />
				<feComposite in="SourceGraphic" in2="blur" operator="over" />
			</filter>
		</defs>

		<!-- Sky Background -->
		<rect width="500" height="420" fill="url(#skyGrad-{archetype.code})" />

		<!-- Sunburst / Celestial Glow -->
		<circle cx="250" cy="140" r="180" fill="url(#sunGlow-{archetype.code})" />

		<!-- Distant Fantasy Kingdom / Castle Silhouette -->
		<g opacity="0.32" fill="#ffffff">
			<!-- Castle Left Spire -->
			<path d="M 60 280 L 60 180 L 80 140 L 100 180 L 100 280 Z" />
			<path d="M 45 280 L 45 210 L 60 190 L 75 210 L 75 280 Z" />
			<polygon points="80,110 75,140 85,140" fill={themeData.wingColor} />
			<!-- Castle Right Towers -->
			<path d="M 400 280 L 400 190 L 420 150 L 440 190 L 440 280 Z" />
			<path d="M 425 280 L 425 220 L 440 200 L 455 220 L 455 280 Z" />
			<polygon points="420,120 415,150 425,150" fill={themeData.wingColor} />
		</g>

		<!-- Golden Stars Floating in Sky -->
		<g fill="#ffeb3b" filter="url(#glowFilter)">
			<!-- Star Top Left -->
			<polygon points="70,55 74,68 87,68 76,76 80,89 70,80 60,89 64,76 53,68 66,68" opacity="0.85" />
			<!-- Star Top Right -->
			<polygon points="430,70 433,79 443,79 435,85 438,94 430,88 422,94 425,85 417,79 427,79" opacity="0.9" />
			<!-- Star Mid -->
			<polygon points="370,160 373,168 382,168 375,173 378,181 370,176 362,181 365,173 358,168 367,168" opacity="0.75" />
			<!-- Star Left Mid -->
			<polygon points="120,130 122,136 128,136 123,140 125,146 120,142 115,146 117,140 112,136 118,136" opacity="0.7" />
		</g>

		<!-- Fluffy Dream Clouds -->
		<g fill="#ffffff" opacity="0.88">
			<!-- Cloud Left -->
			<path d="M 0 310 Q 40 280 90 295 Q 140 260 190 290 Q 230 270 260 310 L 0 310 Z" />
			<!-- Cloud Right -->
			<path d="M 240 310 Q 290 265 350 285 Q 410 250 470 280 Q 495 270 500 290 L 500 310 Z" />
		</g>

		<!-- Floating Magical Books & Sparkles -->
		<g transform="translate(40, 150) rotate(-18)">
			<rect x="0" y="0" width="38" height="26" rx="4" fill="#3b82f6" />
			<path d="M 3 3 L 18 3 L 18 23 L 3 23 Z" fill="#ffffff" />
			<path d="M 20 3 L 35 3 L 35 23 L 20 23 Z" fill="#ffffff" />
			<line x1="19" y1="2" x2="19" y2="24" stroke="#d97706" stroke-width="2" />
		</g>
	</svg>

	<!-- Main Character Stage Composite -->
	<div class="absolute inset-0 flex items-center justify-center">
		<!-- Central Hero Kid Avatar & Pedestal -->
		<div class="relative flex flex-col items-center pt-2">
			<!-- Hero Glowing Aura -->
			<div
				class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 size-48 rounded-full blur-xl"
				style="background: {themeData.accentGlow};"
			></div>

			<!-- Dynamic Hero Avatar Disc -->
			<div class="relative flex size-36 sm:size-40 items-center justify-center rounded-full bg-white shadow-2xl ring-8 ring-amber-300/80">
				<Avatar code={child.avatar} size={110} />

				<!-- Victory Raised Fist Badge -->
				<div class="absolute -top-1 -right-1 flex size-11 items-center justify-center rounded-full bg-gradient-to-tr from-amber-500 to-yellow-300 text-ink shadow-lg ring-3 ring-white">
					<svg viewBox="0 0 24 24" class="size-6 fill-current">
						<path d="M19 10.5V8a2 2 0 0 0-2-2h-1V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v1H7a2 2 0 0 0-2 2v2.5A3.5 3.5 0 0 0 6 17v2a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2v-2a3.5 3.5 0 0 0 1-6.5z" />
					</svg>
				</div>

				<!-- Archetype Special Accessory Badge -->
				{#if archetype.code === 'ksatria_api'}
					<div class="absolute -bottom-2 -left-2 flex size-10 items-center justify-center rounded-full bg-gradient-to-tr from-red-600 to-orange-400 text-white shadow-md ring-2 ring-white">
						<svg viewBox="0 0 24 24" class="size-6 fill-current"><path d="M12 2C9 7 5 9.5 5 14a7 7 0 0 0 14 0c0-4.5-4-7-7-12zm0 16a4 4 0 0 1-4-4c0-2.4 2-3.8 4-6.5 2 2.7 4 4.1 4 6.5a4 4 0 0 1-4 4z" /></svg>
					</div>
				{:else if archetype.code === 'penyihir_angka'}
					<div class="absolute -bottom-2 -left-2 flex size-10 items-center justify-center rounded-full bg-gradient-to-tr from-purple-600 to-indigo-400 text-white shadow-md ring-2 ring-white">
						<svg viewBox="0 0 24 24" class="size-6 fill-current"><path d="M7 2v11h3v9l7-12h-4l4-8z" /></svg>
					</div>
				{:else if archetype.code === 'detektif_elang'}
					<div class="absolute -bottom-2 -left-2 flex size-10 items-center justify-center rounded-full bg-gradient-to-tr from-blue-600 to-sky-400 text-white shadow-md ring-2 ring-white">
						<svg viewBox="0 0 24 24" class="size-6 fill-current"><circle cx="11" cy="11" r="7" stroke="currentColor" stroke-width="2" fill="none" /><line x1="16.5" y1="16.5" x2="22" y2="22" stroke="currentColor" stroke-width="3" /></svg>
					</div>
				{/if}
			</div>
		</div>

		<!-- Cute Companion Mascot: Flying Robot Buddy with Gold Crown & Open Book -->
		<div class="absolute right-3 top-10 sm:right-8 sm:top-8 flex flex-col items-center anim-wiggle">
			<!-- Mascot Floating Glow -->
			<div class="relative flex flex-col items-center">
				<!-- Gold Crown on Mascot -->
				<svg viewBox="0 0 32 20" class="w-8 -mb-1 fill-amber-400 drop-shadow">
					<path d="M 0 18 L 4 4 L 11 12 L 16 0 L 21 12 L 28 4 L 32 18 Z" />
				</svg>

				<!-- Robot Head & Face -->
				<div class="relative flex size-14 items-center justify-center rounded-2xl bg-white shadow-xl ring-3 ring-sky-300">
					<!-- Blue Robot Visor & Happy Eyes -->
					<div class="flex h-7 w-11 items-center justify-around rounded-xl bg-night px-2">
						<div class="size-2 rounded-full bg-sky-400 shadow-[0_0_8px_#38bdf8]"></div>
						<div class="size-2 rounded-full bg-sky-400 shadow-[0_0_8px_#38bdf8]"></div>
					</div>

					<!-- Cute Antennas / Robot Ears -->
					<span class="absolute -left-1.5 size-3 rounded-full bg-sky-400"></span>
					<span class="absolute -right-1.5 size-3 rounded-full bg-sky-400"></span>
				</div>

				<!-- Companion Open Book in Hand -->
				<div class="relative -mt-1.5 flex h-5 w-12 items-center justify-center rounded-md bg-amber-400 shadow-md">
					<div class="h-3 w-5 border-r border-amber-600 bg-white"></div>
					<div class="h-3 w-5 bg-white"></div>
				</div>
			</div>
		</div>
	</div>
</div>
