#!/usr/bin/env node

import fs from "node:fs";

function read(path) {
  return fs.readFileSync(new URL(`../${path}`, import.meta.url), "utf8");
}

function json(path) {
  return JSON.parse(read(path));
}

function expectEqual(label, actual, expected) {
  if (actual !== expected) {
    throw new Error(
      `${label} is ${JSON.stringify(actual)}; expected ${JSON.stringify(expected)}`,
    );
  }
}

function expectIncludes(label, contents, value) {
  if (!contents.includes(value)) {
    throw new Error(`${label} does not contain ${JSON.stringify(value)}`);
  }
}

const version = read("VERSION").trim();
if (!/^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.test(version)) {
  throw new Error(
    `VERSION is not a stable semantic version: ${JSON.stringify(version)}`,
  );
}

for (const project of ["web", "site"]) {
  const manifest = json(`${project}/package.json`);
  const lock = json(`${project}/package-lock.json`);
  expectEqual(`${project}/package.json version`, manifest.version, version);
  expectEqual(`${project}/package-lock.json version`, lock.version, version);
  expectEqual(
    `${project}/package-lock.json root version`,
    lock.packages?.[""]?.version,
    version,
  );
}

const openapi = json("api/openapi.json");
expectEqual("OpenAPI info.version", openapi.info?.version, version);

expectIncludes("CHANGELOG.md", read("CHANGELOG.md"), `## [${version}] - `);
expectIncludes(
  "README.md",
  read("README.md"),
  `ghcr.io/jamiekennedy/local-totp:v${version}`,
);
expectIncludes(
  "site home page",
  read("site/src/pages/index.astro"),
  `v${version}`,
);

for (const path of [
  "site/src/components/InstallTabs.tsx",
  "site/src/pages/docs/api.mdx",
  "site/src/pages/docs/deployment.mdx",
  "site/src/pages/docs/installation.mdx",
]) {
  expectIncludes(path, read(path), version);
}

if (process.env.GITHUB_REF_TYPE === "tag") {
  expectEqual("release tag", process.env.GITHUB_REF_NAME, `v${version}`);
}

console.log(`Version surfaces verified for v${version}.`);
