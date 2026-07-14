import { access, readFile, readdir } from "node:fs/promises";
import { dirname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const repositoryRoot = resolve(packageRoot, "..");
const outputRoot = join(packageRoot, "dist");
const basePath = "/local-totp/";
const canonicalOrigin = "https://jamiekennedy.github.io";

const expectedFiles = [
  ".nojekyll",
  "404.html",
  "index.html",
  "openapi.json",
  "docs/installation/index.html",
  "docs/deployment/index.html",
  "docs/cli/index.html",
  "docs/api/index.html",
];

for (const file of expectedFiles) {
  await access(join(outputRoot, file));
}

async function listHTML(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) files.push(...(await listHTML(path)));
    else if (entry.name.endsWith(".html")) files.push(path);
  }
  return files;
}

function targetFor(pathname) {
  const withoutBase =
    pathname === basePath.slice(0, -1) ? "" : pathname.slice(basePath.length);
  const decoded = decodeURIComponent(withoutBase);
  return join(
    outputRoot,
    decoded.endsWith("/") || decoded === "" ? `${decoded}index.html` : decoded,
  );
}

const htmlFiles = await listHTML(outputRoot);
const htmlByName = new Map();
for (const file of htmlFiles) {
  const html = await readFile(file, "utf8");
  const displayName = relative(outputRoot, file).split(sep).join("/");
  htmlByName.set(displayName, html);
  if (displayName !== "404.html" && !html.includes('rel="canonical"')) {
    throw new Error(`${displayName}: missing canonical URL`);
  }
  if (!html.includes('property="og:title"'))
    throw new Error(`${displayName}: missing Open Graph title`);
  if (!html.includes('name="twitter:card"'))
    throw new Error(`${displayName}: missing Twitter metadata`);

  const attributes = html.matchAll(/(?:href|src)="([^"]+)"/g);
  for (const match of attributes) {
    const value = match[1];
    if (
      !value ||
      value.startsWith("#") ||
      value.startsWith("data:") ||
      value.startsWith("mailto:")
    ) {
      continue;
    }
    const url = new URL(value, `${canonicalOrigin}${basePath}`);
    if (url.origin !== canonicalOrigin) continue;
    if (
      !url.pathname.startsWith(basePath) &&
      url.pathname !== basePath.slice(0, -1)
    ) {
      throw new Error(`${displayName}: path escapes Astro base: ${value}`);
    }
    await access(targetFor(url.pathname));
  }
}

function anchorsFor(html, href) {
  return [...html.matchAll(/<a\b[^>]*>/g)]
    .map(([tag]) => tag)
    .filter((tag) => tag.includes(`href="${href}"`));
}

function hasClasses(tag, classes) {
  const classAttribute = tag.match(/class="([^"]*)"/)?.[1] ?? "";
  const classNames = new Set(classAttribute.split(/\s+/));
  return classes.every((className) => classNames.has(className));
}

const landingHTML = htmlByName.get("index.html");
const installationCTA = anchorsFor(
  landingHTML,
  `${basePath}docs/installation/`,
).find((tag) => hasClasses(tag, ["inline-flex", "bg-primary", "h-11"]));
if (!installationCTA) {
  throw new Error("index.html: installation CTA is missing button styles");
}

const styledGitHubLinks = anchorsFor(
  landingHTML,
  "https://github.com/JamieKennedy/local-totp",
).filter((tag) => tag.includes("inline-flex"));
if (styledGitHubLinks.length < 2) {
  throw new Error("index.html: header or hero GitHub CTA is unstyled");
}

const notFoundHTML = htmlByName.get("404.html");
const returnHome = anchorsFor(notFoundHTML, basePath).find((tag) =>
  hasClasses(tag, ["inline-flex", "bg-primary"]),
);
if (!returnHome) {
  throw new Error("404.html: return-home action is missing button styles");
}

for (const [file, expectedTables] of [
  ["docs/installation/index.html", 2],
  ["docs/cli/index.html", 2],
  ["docs/api/index.html", 2],
]) {
  const html = htmlByName.get(file);
  const tableFrames = html.match(/class="[^"]*\bdoc-table\b[^"]*"/g) ?? [];
  if (tableFrames.length !== expectedTables) {
    throw new Error(
      `${file}: expected ${expectedTables} responsive table frames, found ${tableFrames.length}`,
    );
  }
}

const sourceOpenAPI = JSON.parse(
  await readFile(join(repositoryRoot, "api/openapi.json"), "utf8"),
);
const builtOpenAPI = JSON.parse(
  await readFile(join(outputRoot, "openapi.json"), "utf8"),
);
if (JSON.stringify(sourceOpenAPI) !== JSON.stringify(builtOpenAPI)) {
  throw new Error("dist/openapi.json differs from canonical api/openapi.json");
}

console.log(
  `Verified ${htmlFiles.length} HTML files, internal paths, metadata, and OpenAPI output.`,
);
