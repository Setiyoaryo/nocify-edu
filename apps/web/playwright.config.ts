import { defineConfig, devices } from '@playwright/test';

// Runs against a live stack (API + web). Locally: `make dev` then `npm run test:e2e`.
// BASE_URL can point at staging for smoke runs.
export default defineConfig({
	testDir: 'e2e',
	timeout: 60_000,
	use: { baseURL: process.env.BASE_URL ?? 'http://localhost:5173', trace: 'retain-on-failure' },
	webServer: {
		command: 'npm run dev',
		port: 5173,
		reuseExistingServer: true,
	},
	projects: [
		{ name: 'mobile-360', use: { ...devices['Pixel 7'], viewport: { width: 360, height: 800 } } },
		{ name: 'desktop', use: { ...devices['Desktop Chrome'], viewport: { width: 1280, height: 720 } } }
	]
});
