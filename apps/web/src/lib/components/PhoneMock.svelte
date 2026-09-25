<script lang="ts">
	// A live-HTML miniature of the real child home, used as the landing hero.
	// Crisp at any size, no image weight, and it shows the actual product.
	import Avatar from './Avatar.svelte';
	import Star from 'phosphor-svelte/lib/Star';
	import Flag from 'phosphor-svelte/lib/Flag';
	import MathOperations from 'phosphor-svelte/lib/MathOperations';
	import BookOpenText from 'phosphor-svelte/lib/BookOpenText';
	import House from 'phosphor-svelte/lib/House';
	import Gift from 'phosphor-svelte/lib/Gift';
	import UserCircle from 'phosphor-svelte/lib/UserCircle';
	import Fire from 'phosphor-svelte/lib/Fire';
	import Play from 'phosphor-svelte/lib/Play';
</script>

<div class="relative mx-auto w-[248px] rounded-[2.6rem] bg-night p-2.5 shadow-[0_40px_80px_-24px_rgb(10_10_40/0.55)] sm:w-[272px]" role="img" aria-label="Tampilan aplikasi anak: sapaan, misi dari orang tua, dan pilihan pelajaran">
	<div class="absolute top-2.5 left-1/2 z-10 h-5 w-24 -translate-x-1/2 rounded-b-2xl bg-night" aria-hidden="true"></div>
	<div class="relative h-[500px] overflow-hidden rounded-[2.1rem] bg-canvas text-ink sm:h-[540px]" aria-hidden="true">
		<div class="flex items-center justify-between px-5 pt-3 text-[10px] font-bold"><span>09.41</span><span class="flex gap-1"><span class="h-2 w-3 rounded-sm bg-ink"></span><span class="h-2 w-4 rounded-sm bg-ink"></span></span></div>
		<div class="flex items-center gap-2 px-4 pt-3">
			<Avatar code="cat" size={30} />
			<div class="flex-1 leading-tight"><p class="text-[12px] font-extrabold">Adi</p><p class="text-[9px] text-ink-soft">Level 2</p></div>
			<span class="flex items-center gap-1 rounded-full bg-sun-tint px-2 py-0.5 text-[10px] font-extrabold"><Star size={11} weight="fill" class="text-sun-deep" />12</span>
		</div>
		<div class="pastel-lilac mx-3 mt-3 rounded-2xl p-3">
			<p class="text-[9px] font-bold text-ink-soft">Selamat pagi,</p>
			<p class="text-[17px] leading-tight font-extrabold">Halo, Adi!</p>
			<div class="mt-2 flex gap-1">
				<span class="flex items-center gap-0.5 rounded-full bg-white px-1.5 py-0.5 text-[8.5px] font-bold"><Fire size={9} weight="fill" class="text-tangerine" />3 hari</span>
				<span class="flex items-center gap-0.5 rounded-full bg-white px-1.5 py-0.5 text-[8.5px] font-bold"><Star size={9} weight="fill" class="text-sun-deep" />12 bintang</span>
			</div>
		</div>
		<p class="mx-4 mt-3 flex items-center gap-1 text-[11px] font-extrabold"><Flag size={12} weight="fill" class="text-tangerine" />Misi dari orang tua</p>
		<div class="pastel-sun mx-3 mt-1.5 rounded-2xl p-3">
			<div class="flex items-center gap-2">
				<span class="grid size-7 place-items-center rounded-full bg-white"><Flag size={14} weight="duotone" color="#c9501c" /></span>
				<div class="flex-1"><p class="text-[11px] font-extrabold">Soal cerita</p><p class="text-[8.5px] text-ink-soft">2 dari 5 challenge</p></div>
			</div>
			<div class="mt-2 h-1.5 overflow-hidden rounded-full bg-white"><div class="phone-fill h-full w-2/5 rounded-full bg-tangerine"></div></div>
		</div>
		<p class="mx-4 mt-3 text-[11px] font-extrabold">Pilih pelajaran</p>
		<div class="mx-3 mt-1.5 grid grid-cols-2 gap-2">
			<div class="pastel-grape relative overflow-hidden rounded-2xl p-2.5">
				<span class="grid size-7 place-items-center rounded-full bg-white"><MathOperations size={15} weight="duotone" color="#5b5ce2" /></span>
				<p class="mt-2 text-[11px] font-extrabold">Matematika</p>
				<span class="mt-2 flex items-center justify-center gap-1 rounded-full bg-white py-1 text-[9px] font-extrabold"><Play size={8} weight="fill" />Mulai</span>
			</div>
			<div class="pastel-peach relative overflow-hidden rounded-2xl p-2.5">
				<span class="grid size-7 place-items-center rounded-full bg-white"><BookOpenText size={15} weight="duotone" color="#c9501c" /></span>
				<p class="mt-2 text-[11px] font-extrabold">Bahasa</p>
				<span class="mt-2 flex items-center justify-center gap-1 rounded-full bg-white py-1 text-[9px] font-extrabold"><Play size={8} weight="fill" />Mulai</span>
			</div>
		</div>
		<div class="absolute inset-x-0 bottom-0 grid grid-cols-3 border-t border-line bg-white pt-2 pb-3 text-center text-[9px] font-extrabold text-ink-soft">
			<span class="flex flex-col items-center text-primary"><House size={16} weight="fill" />Main</span>
			<span class="flex flex-col items-center"><Gift size={16} />Hadiah</span>
			<span class="flex flex-col items-center"><UserCircle size={16} />Profil</span>
		</div>
	</div>
</div>

<style>
	.phone-fill {
		transform-origin: left;
		animation: fill-x 1.2s var(--ease-out) 0.9s both;
	}
</style>
