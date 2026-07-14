import type { APIContext } from "astro";
import { documentationNavigation } from "@/lib/navigation";
import { withBase } from "@/lib/paths";

export const prerender = true;

function escapeXML(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

export function GET({ site }: APIContext): Response {
  if (!site) throw new Error("Astro site URL is required");
  const paths = [
    withBase(),
    ...documentationNavigation.map((item) => item.href),
  ];
  const urls = paths
    .map(
      (path) => `<url><loc>${escapeXML(new URL(path, site).href)}</loc></url>`,
    )
    .join("");
  const document = `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${urls}</urlset>`;
  return new Response(document, {
    headers: { "Content-Type": "application/xml; charset=utf-8" },
  });
}
