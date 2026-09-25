<script lang="ts">
	// A light "grown-ups only" check before leaving child mode — not security
	// (the parent account is the boundary), just a guard against accidental exits.
	import LockKey from 'phosphor-svelte/lib/LockKey';
	let { onpass, oncancel }: { onpass: () => void; oncancel: () => void } = $props();
	const a = 6 + Math.floor(Math.random() * 4);
	const b = 6 + Math.floor(Math.random() * 4);
	let answer = $state('');
	let wrong = $state(false);
	let dialog: HTMLDialogElement;
	$effect(() => dialog.showModal());
	function check(e: SubmitEvent) {
		e.preventDefault();
		if (Number(answer) === a * b) onpass();
		else wrong = true;
	}
</script>

<dialog bind:this={dialog} class="m-auto w-[min(92vw,24rem)] rounded-3xl bg-paper p-7 text-ink shadow-[var(--shadow-lift)] backdrop:bg-night/60" oncancel={oncancel} aria-labelledby="gate-title">
	<form onsubmit={check}>
		<span class="grid size-12 place-items-center rounded-2xl bg-primary-tint text-primary-deep"><LockKey size={26} weight="duotone" aria-hidden="true" /></span>
		<h2 id="gate-title" class="mt-4 text-2xl">Khusus orang tua</h2>
		<label class="mt-4 block">
			<span class="label">Berapa {a} × {b}?</span>
			<input class="field text-xl" inputmode="numeric" bind:value={answer} autocomplete="off" />
		</label>
		{#if wrong}<p class="field-error">Belum tepat.</p>{/if}
		<div class="mt-6 flex gap-3">
			<button class="btn btn-primary flex-1">Lanjut</button>
			<button type="button" class="btn btn-quiet flex-1" onclick={oncancel}>Batal</button>
		</div>
	</form>
</dialog>
