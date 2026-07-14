import { readFile } from "node:fs/promises";
import { resolve } from "node:path";

export const prerender = true;

export async function GET(): Promise<Response> {
  const document = await readFile(
    resolve(process.cwd(), "../api/openapi.json"),
    "utf8",
  );
  return new Response(document, {
    headers: { "Content-Type": "application/json; charset=utf-8" },
  });
}
