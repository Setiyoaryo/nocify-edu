<script lang="ts">
	import { get } from '$lib/api';
	import QuestionRenderer from '$lib/questions/QuestionRenderer.svelte';
	import type { Response } from '$lib/types';
	import type { Snippet } from 'svelte';
	import { figures } from '$lib/figures';
	import Figure from '$lib/components/Figure.svelte';

	export type Draft = {
		code: string; skill_id: number | null; type: string; difficulty: number; learning_objective: string;
		source: string; content: string; answer: string; explanation: string; hint: string;
	};
	let {
		value = $bindable(),
		fields = {},
		isNew = false,
		after
	}: { value: Draft; fields?: Record<string, string>; isNew?: boolean; after?: Snippet<[Response | null]> } = $props();

	// Starter payloads per type, so editors see the exact shape the renderer expects.
	const templates: Record<string, { content: object; answer: object }> = {
		single_choice: { content: { prompt: '', options: [{ id: 'a', label: '' }, { id: 'b', label: '' }, { id: 'c', label: '' }] }, answer: { option_id: 'a' } },
		image_choice: { content: { prompt: '', audio_text: '', options: [{ id: 'a', label: '', figures: [{ icon: 'orange', count: 3 }] }, { id: 'b', label: '', figures: [{ icon: 'orange', count: 5 }] }] }, answer: { option_id: 'b' } },
		tap_count: { content: { prompt: '', audio_text: '', symbol: 'orange', color: 'orange', count: 5, choices: [4, 5, 6] }, answer: { value: 5 } },
		number_input: { content: { prompt: '' }, answer: { value: 0 } },
		short_answer: { content: { prompt: '' }, answer: { accepted: [''] } },
		sequence: { content: { prompt: '', items: [{ id: '1', label: '' }, { id: '2', label: '' }, { id: '3', label: '' }] }, answer: { order: ['1', '2', '3'] } },
		matching: { content: { prompt: '', left: [{ id: 'l1', label: '' }, { id: 'l2', label: '' }], right: [{ id: 'r1', label: '' }, { id: 'r2', label: '' }] }, answer: { pairs: { l1: 'r1', l2: 'r2' } } }
	};

	let types = $state<{ code: string; label: string }[]>([]);
	let tree = $state<any>(null);
	get('/admin/question-types').then((r) => (types = r.types));
	get('/admin/curriculum').then((r) => (tree = r));

	function applyTemplate() {
		const t = templates[value.type];
		if (t && isNew) {
			value.content = JSON.stringify(t.content, null, 2);
			value.answer = JSON.stringify(t.answer, null, 2);
		}
	}

	const parsed = $derived.by(() => {
		try {
			return { ok: true as const, content: JSON.parse(value.content) };
		} catch (e: any) {
			return { ok: false as const, error: e.message };
		}
	});
	const skillLabel = (k: any) => {
		const ch = tree.chapters.find((c: any) => c.id === k.chapter_id);
		const g = tree.grades.find((x: any) => x.id === ch?.grade_id);
		const s = tree.subjects.find((x: any) => x.id === ch?.subject_id);
		return `${g?.name}, ${s?.name}: ${ch?.name} › ${k.name}`;
	};
	let previewResponse = $state<Response | null>(null);
</script>

<div class="grid gap-6 xl:grid-cols-2">
	<div class="card space-y-5 p-5 sm:p-6">
		<div class="grid gap-4 sm:grid-cols-2">
			<div>
				<label class="label" for="code">Kode</label>
				<input id="code" class="field" bind:value={value.code} placeholder="Otomatis bila kosong" disabled={!isNew} />
				{#if fields.code}<p class="field-error">{fields.code}</p>{/if}
			</div>
			<div>
				<label class="label" for="type">Tipe soal</label>
				<select id="type" class="field" bind:value={value.type} onchange={applyTemplate}>
					{#each types as t}<option value={t.code}>{t.label} ({t.code})</option>{/each}
				</select>
			</div>
		</div>
		<div>
			<label class="label" for="skill">Skill</label>
			<select id="skill" class="field" bind:value={value.skill_id}>
				<option value={null} disabled>Pilih skill</option>
				{#if tree}{#each tree.skills as k}<option value={k.id}>{skillLabel(k)}</option>{/each}{/if}
			</select>
			{#if fields.skill_id}<p class="field-error">{fields.skill_id}</p>{/if}
		</div>
		<div class="grid gap-4 sm:grid-cols-[8rem_1fr]">
			<div>
				<label class="label" for="diff">Kesulitan</label>
				<select id="diff" class="field" bind:value={value.difficulty}>{#each [1, 2, 3, 4, 5] as d}<option value={d}>{d}</option>{/each}</select>
			</div>
			<div>
				<label class="label" for="lo">Tujuan pembelajaran</label>
				<input id="lo" class="field" bind:value={value.learning_objective} />
				{#if fields.learning_objective}<p class="field-error">{fields.learning_objective}</p>{/if}
			</div>
		</div>
		<div>
			<label class="label" for="content">Konten (JSON)</label>
			<textarea id="content" class="field min-h-56 font-mono text-sm" spellcheck="false" bind:value={value.content}></textarea>
			{#if !parsed.ok}<p class="field-error">JSON tidak valid: {parsed.error}</p>{/if}
			{#if fields.content}<p class="field-error">{fields.content}</p>{/if}
			<p class="mt-1 text-xs text-ink-soft">Benda bergambar memakai <code>figures</code>: <code>{'{"icon":"orange","count":3,"color":"orange"}'}</code>. Ikon yang tersedia ada di daftar di bawah. Gambar lain harus berasal dari Aset yang sudah tayang (<code>/api/v1/media/&lt;id&gt;</code>) dengan teks alt. Emoji tidak diterima.</p>
			<details class="mt-2 text-xs">
				<summary class="cursor-pointer font-bold text-primary-deep">Lihat ikon yang tersedia</summary>
				<div class="mt-2 flex flex-wrap gap-1.5">
					{#each Object.keys(figures) as name}<span class="flex items-center gap-1 rounded-lg bg-paper-deep px-2 py-1"><Figure {name} size={16} />{name}</span>{/each}
				</div>
			</details>
		</div>
		<div>
			<label class="label" for="answer">Jawaban benar (JSON)</label>
			<textarea id="answer" class="field min-h-20 font-mono text-sm" spellcheck="false" bind:value={value.answer}></textarea>
		</div>
		<div>
			<label class="label" for="expl">Penjelasan</label>
			<textarea id="expl" class="field min-h-20" bind:value={value.explanation}></textarea>
			{#if fields.explanation}<p class="field-error">{fields.explanation}</p>{/if}
		</div>
		<div>
			<label class="label" for="hint">Petunjuk (setelah jawaban keliru)</label>
			<input id="hint" class="field" bind:value={value.hint} />
		</div>
		<div>
			<label class="label" for="src">Sumber / catatan revisi</label>
			<input id="src" class="field" bind:value={value.source} />
			{#if fields.source}<p class="field-error">{fields.source}</p>{/if}
		</div>
	</div>

	<aside aria-label="Pratinjau" class="xl:sticky xl:top-6 xl:self-start">
		<p class="label">Pratinjau tampilan anak</p>
		<div class="card p-5 ring-2 ring-primary/20">
			{#if parsed.ok}
				{#key value.type + value.content}
					<QuestionRenderer type={value.type} content={parsed.content} bind:response={previewResponse} band="lower" />
				{/key}
			{:else}
				<p class="text-ink-soft">Perbaiki JSON untuk melihat pratinjau.</p>
			{/if}
		</div>
		<p class="mt-2 text-xs text-ink-soft">Respons pratinjau: <code>{JSON.stringify(previewResponse)}</code></p>
		{@render after?.(previewResponse)}
	</aside>
</div>
