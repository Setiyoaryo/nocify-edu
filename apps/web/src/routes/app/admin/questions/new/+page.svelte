<script lang="ts">
	import { post, ApiError } from '$lib/api';
	import { goto } from '$app/navigation';
	import QuestionEditor, { type Draft } from '../QuestionEditor.svelte';

	let value = $state<Draft>({
		code: '', skill_id: null, type: 'single_choice', difficulty: 2, learning_objective: '', source: '',
		content: JSON.stringify({ prompt: '', options: [{ id: 'a', label: '' }, { id: 'b', label: '' }, { id: 'c', label: '' }] }, null, 2),
		answer: JSON.stringify({ option_id: 'a' }, null, 2), explanation: '', hint: ''
	});
	let fields = $state<Record<string, string>>({});
	let busy = $state(false);

	async function save() {
		busy = true;
		fields = {};
		try {
			const r = await post('/admin/questions', { ...value, content: JSON.parse(value.content), answer: JSON.parse(value.answer) });
			goto(`/app/admin/questions/${r.question.id}`);
		} catch (e) {
			fields = e instanceof ApiError ? (Object.keys(e.fields).length ? e.fields : { content: e.message }) : { content: String(e) };
		}
		busy = false;
	}
</script>

<svelte:head><title>Soal baru | CMS</title></svelte:head>
<a href="/app/admin/questions" class="text-sm font-bold text-primary-deep hover:underline">Kembali ke bank soal</a>
<h1 class="mt-2 text-3xl sm:text-4xl">Soal baru</h1>
<p class="mt-1 text-ink-soft">Soal baru disimpan sebagai Draft dan harus melalui review sebelum tayang.</p>
<div class="mt-8"><QuestionEditor bind:value {fields} isNew /></div>
<button class="btn btn-primary mt-8" disabled={busy} onclick={save}>Simpan draft</button>
