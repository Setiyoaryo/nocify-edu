export const prerender = true;

const pages = ['/', '/features', '/subjects', '/pricing', '/for-parents', '/faq'];

export function GET({ url }) {
	const body = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${pages.map((p) => `  <url><loc>${url.origin}${p}</loc></url>`).join('\n')}
</urlset>`;
	return new Response(body, { headers: { 'Content-Type': 'application/xml' } });
}
