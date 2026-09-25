import { expect, test, type Page } from '@playwright/test';

// §48 critical flow through the real UI:
// Register → Create child → Start challenge → Answer → Complete → Result →
// Parent progress → Subscribe → (signed mock webhook) → Subscription active.

async function answerCurrent(page: Page, attempt: number) {
	const radios = page.locator('fieldset input[type=radio]');
	const radioButtons = page.locator('[role=radiogroup] [role=radio]');
	const keypad = page.getByRole('button', { name: String(attempt + 1), exact: true });
	const shortAnswer = page.getByLabel('Tulis jawabanmu');
	if (await radios.count()) {
		// Tap the visible answer tile, as a child would.
		await page.locator('fieldset label').nth(attempt % (await radios.count())).click();
	} else if (await radioButtons.count()) {
		await radioButtons.nth(attempt % (await radioButtons.count())).click();
	} else if (await shortAnswer.count()) {
		await shortAnswer.fill(attempt ? 'jawaban lain' : 'jawaban');
	} else if (await page.getByLabel('Pilihan', { exact: true }).count()) {
		// sequence: tap remaining items in order
		const group = page.getByLabel('Pilihan', { exact: true });
		if (attempt) await page.getByRole('button', { name: /ketuk untuk melepas/ }).first().click();
		while (await group.getByRole('button').count()) await group.getByRole('button').first().click();
	} else if (await page.getByText('Ketuk satu di kiri').count()) {
		// matching: pair left i with right i (or shifted on retry)
		const cols = page.locator('.grid-cols-2 > div');
		const n = await cols.nth(0).getByRole('button').count();
		for (let i = 0; i < n; i++) {
			await cols.nth(0).getByRole('button').nth(i).click();
			await cols.nth(1).getByRole('button').nth((i + attempt) % n).click();
		}
	} else if (await keypad.count()) {
		await keypad.click();
	}
}

test('parent registers, child plays a challenge, parent subscribes', async ({ page }) => {
	const email = `e2e-${Date.now()}-${Math.random().toString(36).slice(2, 6)}@example.test`;

	await page.goto('/register');
	await expect(page.getByRole('button', { name: 'Buat akun' })).toBeEnabled(); // hydrated
	await page.locator('#name').fill('Ibu Uji');
	await page.locator('#email').fill(email);
	await page.locator('#password').fill('rahasia123');
	await page.getByRole('checkbox').check();
	await page.getByRole('button', { name: 'Buat akun' }).click();

	await expect(page.getByRole('heading', { name: 'Sekarang, profil anak' })).toBeVisible();
	await page.locator('#nickname').fill('Adi');
	await page.locator('#grade').selectOption({ label: 'SD – Kelas 1' });
	await page.getByRole('button', { name: 'Simpan dan mulai bermain' }).click();

	await expect(page.getByRole('heading', { name: 'Halo, Adi!' })).toBeVisible();
	await page.getByRole('button', { name: /Matematika/ }).click();

	const finished = page.getByRole('link', { name: 'Selesai' });
	const next = page.getByRole('button', { name: /^(Lanjut|Lihat hasil)$/ });
	const retry = page.getByRole('button', { name: 'Coba lagi' });
	const check = page.getByRole('button', { name: 'Cek jawaban' });
	for (let guard = 0; guard < 40; guard++) {
		await expect(finished.or(next).or(retry).or(check).first()).toBeVisible();
		if (await finished.isVisible()) break;
		if (await next.isVisible()) {
			await next.click();
			continue;
		}
		const attempt = (await retry.isVisible()) ? 1 : 0;
		await answerCurrent(page, attempt);
		await (attempt ? retry : check).click();
		await expect(next.or(retry).first()).toBeVisible();
	}

	await expect(page.getByText(/^\d+\/\d+$/)).toBeVisible(); // correct / total
	await expect(page.getByText('Langkah Pertama')).toBeVisible(); // first-challenge badge

	// Leave child mode through the parent gate
	await page.getByRole('link', { name: 'Selesai' }).click();
	await page.getByRole('button', { name: 'Keluar ke area orang tua' }).click();
	const q = await page.getByText(/Berapa \d+ × \d+\?/).textContent();
	const [, a, b] = q!.match(/(\d+) × (\d+)/)!;
	await page.getByRole('dialog').getByRole('textbox').fill(String(Number(a) * Number(b)));
	await page.getByRole('button', { name: 'Lanjut' }).click();

	await expect(page.getByRole('heading', { name: 'Ibu', exact: true })).toBeVisible();
	await page.getByRole('link', { name: /Lihat laporan/ }).click();
	await expect(page.getByRole('heading', { name: 'Per pelajaran dan bab' })).toBeVisible();

	// Parent turns a skill into a mission; the child sees it first on their home.
	await page.getByRole('tab', { name: /Misi/ }).click();
	const skill = page.locator('select').first();
	await skill.selectOption({ index: 1 });
	await page.getByRole('button', { name: 'Buat misi' }).click();
	await expect(page.getByRole('button', { name: /Batalkan misi/ })).toBeVisible();
	await page.getByRole('link', { name: /Main sebagai Adi/ }).click();
	await expect(page.getByRole('heading', { name: 'Misi dari orang tua' })).toBeVisible();

	if (process.env.E2E_PAYMENTS === 'mock') {
		await page.goto('/app/parent/subscription');
		await page.getByRole('button', { name: 'Berlangganan sekarang' }).click();
		await page.getByRole('button', { name: 'Simulasikan pembayaran berhasil' }).click();
		await expect(page.getByText('Pembayaran diterima. Langganan Anda aktif.')).toBeVisible();
	}
});

test('marketing pages have no horizontal overflow', async ({ page }) => {
	for (const path of ['/', '/pricing', '/subjects', '/features', '/for-parents', '/faq', '/login', '/register']) {
		await page.goto(path);
		const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
		expect(overflow, `${path} overflows by ${overflow}px`).toBeLessThanOrEqual(0);
	}
});
