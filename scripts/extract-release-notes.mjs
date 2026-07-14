#!/usr/bin/env node

import fs from "node:fs";

const read = (path) =>
  fs.readFileSync(new URL(`../${path}`, import.meta.url), "utf8");

const version = read("VERSION").trim();
const changelog = read("CHANGELOG.md");
const heading = `## [${version}] - `;
const headingStart = changelog.indexOf(heading);

if (headingStart === -1) {
  throw new Error(`CHANGELOG.md has no release heading for ${version}`);
}

const contentStart = changelog.indexOf("\n", headingStart) + 1;
const remaining = changelog.slice(contentStart);
const nextHeading = remaining.search(/^## \[/m);
const notes = (
  nextHeading === -1 ? remaining : remaining.slice(0, nextHeading)
).trim();

if (!notes) {
  throw new Error(`CHANGELOG.md has no release notes for ${version}`);
}

process.stdout.write(`${notes}\n`);
