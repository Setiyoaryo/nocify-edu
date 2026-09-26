import { expect, test } from '@playwright/test';

// Guards against layout regressions: no page may scroll sideways at common
// phone, tablet and desktop widths (§31.3). Runs once (desktop project).
const widths = [360, 768, 1024, 1440];

test('app pages never overflow horizontally', async ({ page }, info) => {
	test.skip(info.project.name !== 'desktop', 'widths are set explicitly');
	const email = `resp-${Date.now()}@example.test`;
	await page.goto('/register');
	await expect(page.getByRole('button', { name: 'Buat akun' })).toBeEnabled();
	await page.locator('#name').fill('Ibu Uji');
	await page.locator('#email').fill(email);
	await page.locator('#password').fill('rahasia123');
	await page.getByRole('checkbox').check();
	await page.getByRole('button', { name: 'Buat akun' }).click();
	await page.locator('#nickname').fill('Adi');
	await page.locator('#grade').selectOption({ label: 'SD – Kelas 1' });
	await page.getByRole('button', { name: 'Simpan dan mulai bermain' }).click();
	await expect(page.getByRole('heading', { name: 'Halo, Adi!' })).toBeVisible();
	const childUrl = page.url();
	const cid = childUrl.split('/').pop();
	await page.getByRole('button', { name: /^Matematika/ }).click();
	await page.waitForURL(/challenge/);
	const challengeUrl = page.url();

	const paths = ['/app/parent', `/app/parent/children/${cid}`, '/app/parent/children', '/app/parent/subscription', childUrl, `${childUrl}/card`, challengeUrl];
	for (const w of widths) {
		await page.setViewportSize({ width: w, height: 900 });
		for (const p of paths) {
			await page.goto(p);
			await page.waitForLoadState('networkidle');
			const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
			expect(overflow, `${p} overflows by ${overflow}px at ${w}px`).toBeLessThanOrEqual(0);
		}
	}
});
