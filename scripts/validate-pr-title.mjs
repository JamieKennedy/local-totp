const title = process.argv.slice(2).join(" ").trim();
const conventional =
  /^(feat|fix|docs|refactor|test|build|ci|chore|perf|revert)(\((api|vault|totp|storage|web|cli|docker|release)\))?!?: [a-z0-9][^.!?]*$/;

if (title.length === 0) {
  console.error("A pull request title is required.");
  process.exit(1);
}
if (title.length > 72) {
  console.error(`Pull request titles must be at most 72 characters (received ${title.length}).`);
  process.exit(1);
}
if (!conventional.test(title)) {
  console.error(
    "Use Conventional Commit form: type(scope)!: imperative summary (lowercase summary, no trailing punctuation).",
  );
  process.exit(1);
}

console.log(`Valid pull request title: ${title}`);
