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
					user: {
						id: '00000000-0000-0000-0000-000000000001',
						email: 'parent@eduplay.test',
						display_name: 'Orang Tua',
						role: 'parent',
						created_at: new Date().toISOString()
					}
				})
			});
		});
		await page.route('**/me', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					user: {
						id: '00000000-0000-0000-0000-000000000001',
						email: 'parent@eduplay.test',
						display_name: 'Orang Tua',
						role: 'parent',
						created_at: new Date().toISOString()
					}
				})
			});
		});
		await page.route('**/children', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ children: [] })
			});
		});
		await page.route('**/subscription', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ subscription: { active: true, plan: 'family', expires_at: '2099-01-01T00:00:00Z' } })
			});
		});

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

		// Enter 6 digit OTP
		for (let i = 0; i < 6; i++) {
			await page.locator(`#otp-${i}`).fill(String(i + 1));
		}
		await page.getByRole('button', { name: 'Verifikasi & Masuk' }).click();

		// Redirects to /app/parent
		await expect(page).toHaveURL(/\/app\/parent/);
		await expect(page.getByRole('heading', { level: 1 })).toBeVisible();

		// Verify "Mode anak" is NOT in the layout (sidebar and mobile header)
		await expect(page.getByRole('link', { name: 'Mode anak' })).toHaveCount(0);
	});

	test('child logs in with WhatsApp OTP into kid friendly play environment', async ({ page }) => {
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
					role: 'child',
					user: {
						id: '00000000-0000-0000-0000-000000000001',
						email: 'parent@eduplay.test',
						display_name: 'Keluarga EduPlay',
						role: 'parent',
						created_at: new Date().toISOString()
					},
					children: [
						{
							id: '11111111-1111-1111-1111-111111111111',
							nickname: 'Petualang',
							grade_id: 1,
							grade_name: 'SD Kelas 1',
							ux_band: 'sd12',
							avatar: 'lion',
							has_pin: false,
							xp: 150,
							level: 2,
							last_activity: null
						}
					]
				})
			});
		});
		await page.route('**/me', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					user: {
						id: '00000000-0000-0000-0000-000000000001',
						email: 'parent@eduplay.test',
						display_name: 'Keluarga EduPlay',
						role: 'parent',
						created_at: new Date().toISOString()
					}
				})
			});
		});
		await page.route('**/children/*/home', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					child: {
						id: '11111111-1111-1111-1111-111111111111',
						nickname: 'Petualang',
						grade_id: 1,
						grade_name: 'SD Kelas 1',
						ux_band: 'sd12',
						avatar: 'lion',
						has_pin: false,
						xp: 150,
						level: 2,
						last_activity: null
					},
					stars: 5,
					streak: 1
				})
			});
		});

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

		for (let i = 0; i < 6; i++) {
			await page.locator(`#otp-${i}`).fill(String(i + 1));
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
		await page.getByRole('tab', { name: 'Anak' }).click();
		await expect(page.getByRole('heading', { name: /Hai Petualang/i })).toBeVisible();

		// Switch back to Orang Tua
		await page.getByRole('tab', { name: /Orang Tua/i }).click();
		await expect(page.getByRole('heading', { name: 'Masuk sebagai Orang Tua' })).toBeVisible();

		// Click "Ganti peran" to go back to initial choice cards
		await page.getByRole('button', { name: 'Ganti peran' }).click();
		await expect(page.getByRole('heading', { name: 'Masuk sebagai siapa?' })).toBeVisible();
	});

	test('child cannot log in if account has not been created by parent', async ({ page }) => {
		await page.route('**/auth/otp/send', async (route) => {
			await route.fulfill({
				status: 404,
				contentType: 'application/json',
				body: JSON.stringify({
					error: 'Nomor WhatsApp belum terdaftar. Minta Ayah atau Bunda mendaftar dan membuat profil anak terlebih dahulu.'
				})
			});
		});

		await page.goto('/login');

		// Select Anak mode
		await page.getByRole('button', { name: /Ruang Bermain/i }).click();

		// Fill unregistered phone
		await page.locator('#phone').fill('089999999999');
		await page.getByRole('button', { name: /Kirim Kode ke Ayah \/ Bunda/i }).click();

		// Error banner explaining parent account requirement appears
		await expect(page.getByText(/Anak hanya bisa masuk jika akun dan profil anak sudah dibuat oleh orang tua/i)).toBeVisible();

		// Has quick button to switch to Mode Orang Tua
		const parentModeBtn = page.getByRole('button', { name: /Masuk Mode Orang Tua/i });
		await expect(parentModeBtn).toBeVisible();
		await parentModeBtn.click();

		// Successfully switched to parent mode
		await expect(page.getByRole('heading', { name: 'Masuk sebagai Orang Tua' })).toBeVisible();
	});

	test('direct URL with role parameter initializes proper mode and back button works', async ({ page }) => {
		await page.goto('/login?role=child');
		await expect(page.getByRole('heading', { name: /Hai Petualang/i })).toBeVisible();

		// Browser back to /login goes back to role selection
		await page.getByRole('button', { name: 'Ganti peran' }).click();
		await expect(page.getByRole('heading', { name: 'Masuk sebagai siapa?' })).toBeVisible();

		// Browser forward goes back to child role
		await page.goBack();
		await expect(page.getByRole('heading', { name: /Hai Petualang/i })).toBeVisible();
	});

	test('handles mobile SMS code autofill in single input correctly', async ({ page }) => {
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

		await page.goto('/login?role=parent');
		await page.locator('#phone').fill('081234567890');
		await page.getByRole('button', { name: /Kirim Kode OTP via WhatsApp/i }).click();
		await expect(page.getByRole('heading', { name: 'Verifikasi Kode OTP' })).toBeVisible();

		// Simulate mobile SMS autofill inserting full 6-digit string into otp-0
		await page.locator('#otp-0').fill('654321');

		// All 6 digit inputs should be populated
		await expect(page.locator('#otp-0')).toHaveValue('6');
		await expect(page.locator('#otp-1')).toHaveValue('5');
		await expect(page.locator('#otp-2')).toHaveValue('4');
		await expect(page.locator('#otp-3')).toHaveValue('3');
		await expect(page.locator('#otp-4')).toHaveValue('2');
		await expect(page.locator('#otp-5')).toHaveValue('1');

		// Submit button is now enabled
		const submitBtn = page.getByRole('button', { name: 'Verifikasi & Masuk' });
		await expect(submitBtn).toBeEnabled();
	});

	test('renders without horizontal overflow on narrow mobile screens (320px)', async ({ page }) => {
		await page.setViewportSize({ width: 320, height: 640 });
		await page.goto('/login');
		let scrollWidth = await page.evaluate(() => document.documentElement.scrollWidth);
		expect(scrollWidth).toBeLessThanOrEqual(320);

		await page.getByRole('button', { name: /Area Orang Tua/i }).click();
		scrollWidth = await page.evaluate(() => document.documentElement.scrollWidth);
		expect(scrollWidth).toBeLessThanOrEqual(320);

		await page.getByRole('tab', { name: 'Anak' }).click();
		scrollWidth = await page.evaluate(() => document.documentElement.scrollWidth);
		expect(scrollWidth).toBeLessThanOrEqual(320);
	});
});
