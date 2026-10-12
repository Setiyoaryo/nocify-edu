import { expect, test } from '@playwright/test';

test.describe('WhatsApp OTP Registration Flow', () => {
	test('renders WhatsApp registration form for parent with clean styling and no emojis', async ({ page }) => {
		await page.goto('/register');

		await expect(page.getByRole('heading', { name: 'Buat akun orang tua' })).toBeVisible();

		// Check name input
		const nameInput = page.locator('#name');
		await expect(nameInput).toBeVisible();

		// Check phone input with +62 prefix
		const phoneInput = page.locator('#phone');
		await expect(phoneInput).toBeVisible();
		await expect(page.getByText('+62')).toBeVisible();

		// Check consent checkbox
		const consentCheckbox = page.getByRole('checkbox');
		await expect(consentCheckbox).toBeVisible();

		// Check submit button
		const submitButton = page.getByRole('button', { name: /Daftar dengan WhatsApp/i });
		await expect(submitButton).toBeVisible();

		// Verify no emoji in page text
		const content = await page.content();
		const emojiRegex = /[\u{1F300}-\u{1F9FF}\u{2600}-\u{26FF}\u{2700}-\u{27BF}]/u;
		expect(emojiRegex.test(content)).toBe(false);
	});

	test('shows validation errors when submitting with empty or missing fields', async ({ page }) => {
		await page.goto('/register');

		// Click without filling anything
		await page.getByRole('button', { name: /Daftar dengan WhatsApp/i }).click();

		await expect(page.getByText('Nama lengkap atau panggilan wajib diisi.')).toBeVisible();
		await expect(page.getByText('Nomor WhatsApp wajib diisi.')).toBeVisible();
		await expect(page.getByText('Persetujuan orang tua/wali diperlukan untuk mendaftar.')).toBeVisible();

		// Fill name only
		await page.locator('#name').fill('Bunda Dewi');
		await page.getByRole('button', { name: /Daftar dengan WhatsApp/i }).click();
		await expect(page.getByText('Nomor WhatsApp wajib diisi.')).toBeVisible();
		await expect(page.getByText('Persetujuan orang tua/wali diperlukan untuk mendaftar.')).toBeVisible();

		// Fill phone but leave consent unchecked
		await page.locator('#phone').fill('081234567890');
		await page.getByRole('button', { name: /Daftar dengan WhatsApp/i }).click();
		await expect(page.getByText('Persetujuan orang tua/wali diperlukan untuk mendaftar.')).toBeVisible();
	});

	test('completes registration flow: send OTP -> verify OTP -> redirect to onboarding', async ({ page }) => {
		let sendPayload: any = null;
		let verifyPayload: any = null;

		await page.route('**/auth/otp/send', async (route) => {
			sendPayload = route.request().postDataJSON();
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					status: 'ok',
					phone: '+6281234567890',
					message: 'Kode OTP berhasil dikirim ke WhatsApp.'
				})
			});
		});

		await page.route('**/auth/otp/verify', async (route) => {
			verifyPayload = route.request().postDataJSON();
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				headers: {
					'Set-Cookie': 'eduplay_session=mocksession; Path=/; HttpOnly'
				},
				body: JSON.stringify({
					role: 'parent',
					is_new_user: true,
					user: {
						id: '00000000-0000-0000-0000-000000000002',
						email: 'wa-6281234567890@eduplay.local',
						display_name: 'Bunda Dewi',
						role: 'parent',
						created_at: new Date().toISOString()
					}
				})
			});
		});

		// Mock session endpoint for subsequent redirect
		await page.route('**/auth/session', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					user: {
						id: '00000000-0000-0000-0000-000000000002',
						email: 'wa-6281234567890@eduplay.local',
						display_name: 'Bunda Dewi',
						role: 'parent',
						created_at: new Date().toISOString()
					}
				})
			});
		});

		await page.goto('/register');

		await page.locator('#name').fill('Bunda Dewi');
		await page.locator('#phone').fill('081234567890');
		await page.getByRole('checkbox').check();

		await page.getByRole('button', { name: /Daftar dengan WhatsApp/i }).click();

		// Verify OTP step is now visible
		await expect(page.getByRole('heading', { name: 'Verifikasi Kode OTP' })).toBeVisible();
		await expect(page.getByText('+6281234567890')).toBeVisible();
		expect(sendPayload).toEqual({
			phone: '081234567890',
			role: 'parent'
		});

		// Fill 6 digit OTP
		for (let i = 0; i < 6; i++) {
			await page.locator(`#otp-${i}`).fill(String(i + 1));
		}

		await page.getByRole('button', { name: 'Verifikasi & Buat Akun' }).click();

		// Verify verify payload contains display_name, phone, code, role
		expect(verifyPayload).toEqual({
			phone: '+6281234567890',
			code: '123456',
			role: 'parent',
			display_name: 'Bunda Dewi'
		});

		// Expect navigation to child creation welcome page
		await page.waitForURL('**/app/parent/children/new?welcome=1');
	});

	test('allows editing phone or name by clicking change button on OTP step', async ({ page }) => {
		await page.route('**/auth/otp/send', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					status: 'ok',
					phone: '+6281234567890',
					message: 'Kode OTP berhasil dikirim ke WhatsApp.'
				})
			});
		});

		await page.goto('/register');

		await page.locator('#name').fill('Ayah Joko');
		await page.locator('#phone').fill('081234567890');
		await page.getByRole('checkbox').check();
		await page.getByRole('button', { name: /Daftar dengan WhatsApp/i }).click();

		await expect(page.getByRole('heading', { name: 'Verifikasi Kode OTP' })).toBeVisible();

		// Click "Ubah nama atau nomor WhatsApp"
		await page.getByRole('button', { name: /Ubah nama atau nomor WhatsApp/i }).click();

		// Form step is back and values are preserved
		await expect(page.locator('#name')).toHaveValue('Ayah Joko');
		await expect(page.locator('#phone')).toHaveValue('081234567890');
	});

	test('handles mobile SMS autofill on OTP input', async ({ page }) => {
		await page.route('**/auth/otp/send', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					status: 'ok',
					phone: '+6281234567890',
					message: 'Kode OTP berhasil dikirim ke WhatsApp.'
				})
			});
		});

		await page.goto('/register');

		await page.locator('#name').fill('Ayah Joko');
		await page.locator('#phone').fill('081234567890');
		await page.getByRole('checkbox').check();
		await page.getByRole('button', { name: /Daftar dengan WhatsApp/i }).click();

		await expect(page.getByRole('heading', { name: 'Verifikasi Kode OTP' })).toBeVisible();

		// Simulate autofill inserting 6 digits into otp-0
		await page.locator('#otp-0').fill('987654');

		await expect(page.locator('#otp-0')).toHaveValue('9');
		await expect(page.locator('#otp-1')).toHaveValue('8');
		await expect(page.locator('#otp-2')).toHaveValue('7');
		await expect(page.locator('#otp-3')).toHaveValue('6');
		await expect(page.locator('#otp-4')).toHaveValue('5');
		await expect(page.locator('#otp-5')).toHaveValue('4');

		const verifyBtn = page.getByRole('button', { name: 'Verifikasi & Buat Akun' });
		await expect(verifyBtn).toBeEnabled();
	});

	test('renders without horizontal overflow on narrow mobile screens (320px)', async ({ page }) => {
		await page.setViewportSize({ width: 320, height: 640 });
		await page.goto('/register');

		let scrollWidth = await page.evaluate(() => document.documentElement.scrollWidth);
		expect(scrollWidth).toBeLessThanOrEqual(320);

		// Trigger OTP step
		await page.route('**/auth/otp/send', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					status: 'ok',
					phone: '+6281234567890',
					message: 'Kode OTP berhasil dikirim ke WhatsApp.'
				})
			});
		});

		await page.locator('#name').fill('Bunda Dewi');
		await page.locator('#phone').fill('081234567890');
		await page.getByRole('checkbox').check();
		await page.getByRole('button', { name: /Daftar dengan WhatsApp/i }).click();

		await expect(page.getByRole('heading', { name: 'Verifikasi Kode OTP' })).toBeVisible();
		scrollWidth = await page.evaluate(() => document.documentElement.scrollWidth);
		expect(scrollWidth).toBeLessThanOrEqual(320);
	});

	test('displays error message when OTP is invalid or expired', async ({ page }) => {
		await page.route('**/auth/otp/send', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					status: 'ok',
					phone: '+6281234567890',
					message: 'Kode OTP berhasil dikirim ke WhatsApp.'
				})
			});
		});

		await page.route('**/auth/otp/verify', async (route) => {
			await route.fulfill({
				status: 401,
				contentType: 'application/json',
				body: JSON.stringify({
					error: 'invalid_otp',
					message: 'Kode OTP salah atau telah kadaluarsa.'
				})
			});
		});

		await page.goto('/register');
		await page.locator('#name').fill('Ayah Budi');
		await page.locator('#phone').fill('081234567890');
		await page.getByRole('checkbox').check();
		await page.getByRole('button', { name: /Daftar dengan WhatsApp/i }).click();

		await expect(page.getByRole('heading', { name: 'Verifikasi Kode OTP' })).toBeVisible();

		for (let i = 0; i < 6; i++) {
			await page.locator(`#otp-${i}`).fill(String(i + 1));
		}

		await page.getByRole('button', { name: 'Verifikasi & Buat Akun' }).click();

		await expect(page.getByText('Kode OTP salah atau telah kadaluarsa.')).toBeVisible();
	});

	test('redirects existing parent to /app/parent dashboard when is_new_user is false', async ({ page }) => {
		await page.route('**/auth/otp/send', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					status: 'ok',
					phone: '+6281234567890',
					message: 'Kode OTP berhasil dikirim ke WhatsApp.'
				})
			});
		});

		await page.route('**/auth/otp/verify', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				headers: {
					'Set-Cookie': 'eduplay_session=mocksession; Path=/; HttpOnly'
				},
				body: JSON.stringify({
					role: 'parent',
					is_new_user: false,
					user: {
						id: '00000000-0000-0000-0000-000000000001',
						email: 'parent@eduplay.test',
						display_name: 'Ayah Budi',
						role: 'parent',
						created_at: new Date().toISOString()
					}
				})
			});
		});

		await page.route('**/auth/session', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					user: {
						id: '00000000-0000-0000-0000-000000000001',
						email: 'parent@eduplay.test',
						display_name: 'Ayah Budi',
						role: 'parent',
						created_at: new Date().toISOString()
					}
				})
			});
		});

		await page.goto('/register');
		await page.locator('#name').fill('Ayah Budi');
		await page.locator('#phone').fill('081234567890');
		await page.getByRole('checkbox').check();
		await page.getByRole('button', { name: /Daftar dengan WhatsApp/i }).click();

		await expect(page.getByRole('heading', { name: 'Verifikasi Kode OTP' })).toBeVisible();

		for (let i = 0; i < 6; i++) {
			await page.locator(`#otp-${i}`).fill(String(i + 1));
		}

		await page.getByRole('button', { name: 'Verifikasi & Buat Akun' }).click();

		await page.waitForURL('**/app/parent');
	});
});
