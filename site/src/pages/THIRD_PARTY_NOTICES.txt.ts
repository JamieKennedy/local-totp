import { readFile } from "node:fs/promises";
import { resolve } from "node:path";

export const prerender = true;

export async function GET(): Promise<Response> {
  return new Response(
    await readFile(resolve(process.cwd(), "../THIRD_PARTY_NOTICES.md"), "utf8"),
    {
      headers: { "Content-Type": "text/plain; charset=utf-8" },
    },
  );
}
