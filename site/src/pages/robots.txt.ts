import type { APIContext } from "astro";
import { withBase } from "@/lib/paths";

export const prerender = true;

export function GET({ site }: APIContext): Response {
  if (!site) throw new Error("Astro site URL is required");
  const sitemap = new URL(withBase("sitemap.xml"), site);
  return new Response(`User-agent: *\nAllow: /\nSitemap: ${sitemap}\n`, {
    headers: { "Content-Type": "text/plain; charset=utf-8" },
  });
}
