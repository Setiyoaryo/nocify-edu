import adapter from '@sveltejs/adapter-node';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';

const api = process.env.API_URL ?? 'http://localhost:8080';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			adapter: adapter()
		})
	],
	server: {
		// In development the Go API sits behind the Vite proxy so the browser
		// sees one origin, exactly like Traefik in production.
		proxy: { '/api': { target: api, changeOrigin: false } },
		fs: { allow: ['../../packages'] }
	},
	test: {
		include: ['src/**/*.test.ts']
	}
});
