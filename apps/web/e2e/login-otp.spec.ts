import { expect, test } from '@playwright/test';

test.describe('WhatsApp OTP Login Flow', () => {
	test('displays role choice first (Orang Tua vs Anak) before inputs', async ({ page }) => {
		await page.goto('/login');

		// Choice title
		await expect(page.getByRole('heading', { name: 'Masuk sebagai siapa?' })).toBeVisible();

		// Two distinct role cards
		const parentCard = page.getByRole('button', { name: /Area Orang Tua/i });
		const childCard = page.getByRole('button', { name: /Ruang Bermain/i });

		await expect(parentCard).toBeVisible();
		await expect(childCard).toBeVisible();

		// Phone and password inputs should not be visible yet
		await expect(page.locator('#phone')).toHaveCount(0);
		await expect(page.locator('#password')).toHaveCount(0);
	});

	test('parent logs in with WhatsApp OTP into parent dashboard without Mode Anak button', async ({ page }) => {
		await page.goto('/login');

		// Select Orang Tua
		await page.getByRole('button', { name: /Area Orang Tua/i }).click();

		// Phone input should now be visible with +62 prefix
		await expect(page.locator('#phone')).toBeVisible();
		await expect(page.getByText('+62')).toBeVisible();
		await expect(page.getByRole('button', { name: /Kirim Kode OTP via WhatsApp/i })).toBeVisible();

		// Fill WhatsApp phone number
		await page.locator('#phone').fill('081234567890');
		await page.getByRole('button', { name: /Kirim Kode OTP via WhatsApp/i }).click();

		// OTP screen appears
		await expect(page.getByRole('heading', { name: 'Verifikasi Kode OTP' })).toBeVisible();
		await expect(page.getByText('+6281234567890')).toBeVisible();

		// Enter 6 digit OTP (or use demo code)
		for (let i = 0; i < 6; i++) {
			await page.locator(`#otp-${i}`).fill(String(i === 0 ? 1 : i === 1 ? 2 : i === 2 ? 3 : i === 3 ? 4 : i === 4 ? 5 : 6));
		}
		await page.getByRole('button', { name: 'Verifikasi & Masuk' }).click();

		// Redirects to /app/parent
		await expect(page).toHaveURL(/\/app\/parent/);
		await expect(page.getByRole('heading', { level: 1 })).toBeVisible();

		// Verify "Mode anak" is NOT in the layout (sidebar and mobile header)
		await expect(page.getByRole('link', { name: 'Mode anak' })).toHaveCount(0);
	});

	test('child logs in with WhatsApp OTP into kid friendly play environment', async ({ page }) => {
		await page.goto('/login');

		// Select Anak
		await page.getByRole('button', { name: /Ruang Bermain/i }).click();

		// Kid friendly heading and elements
		await expect(page.getByRole('heading', { name: /Hai Petualang/i })).toBeVisible();
		await expect(page.getByText(/Nomor WhatsApp Ayah \/ Bunda/i)).toBeVisible();

		// Fill WhatsApp phone
		await page.locator('#phone').fill('081234567890');
		await page.getByRole('button', { name: /Kirim Kode ke Ayah \/ Bunda/i }).click();

		// OTP screen appears in kid friendly tone
		await expect(page.getByRole('heading', { name: /Ketik Kode Rahasia/i })).toBeVisible();

		const fillBtn = page.getByRole('button', { name: /Isi otomatis/i });
		if (await fillBtn.isVisible()) {
			await fillBtn.click();
		} else {
			for (let i = 0; i < 6; i++) {
				await page.locator(`#otp-${i}`).fill(String(i + 1));
			}
		}
		await page.getByRole('button', { name: /Ayo Masuk & Main/i }).click();

		// Enters directly into child play view (/app/play/...) or shows child picker
		await expect(page).toHaveURL(/\/app\/play/);
	});

	test('allows smooth role switching from tab bar without losing state', async ({ page }) => {
		await page.goto('/login');

		// Select Orang Tua first
		await page.getByRole('button', { name: /Area Orang Tua/i }).click();
		await expect(page.getByRole('heading', { name: 'Masuk sebagai Orang Tua' })).toBeVisible();

		// Switch to Anak via top tab bar
		await page.getByRole('tab', { name: /Anak ⭐/i }).click();
		await expect(page.getByRole('heading', { name: /Hai Petualang/i })).toBeVisible();

		// Switch back to Orang Tua
		await page.getByRole('tab', { name: /Orang Tua/i }).click();
		await expect(page.getByRole('heading', { name: 'Masuk sebagai Orang Tua' })).toBeVisible();

		// Click "Ganti peran" to go back to initial choice cards
		await page.getByRole('button', { name: 'Ganti peran' }).click();
		await expect(page.getByRole('heading', { name: 'Masuk sebagai siapa?' })).toBeVisible();
	});
});
