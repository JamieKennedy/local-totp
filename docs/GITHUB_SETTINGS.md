# GitHub release settings

Repository settings cannot be fully expressed in source control. Apply this configuration during the public cutover while signed in as `JamieKennedy`.

## Before changing visibility

- Merge only release-approved Dependabot updates after fresh green checks.
- Merge the owner-reviewed release pull request and record the exact green `main` SHA.
- Confirm there are no critical or high Dependabot alerts.
- Remove the obsolete `v0.1.0` GitHub Release, remote/local tag, and GHCR package version. Do not rewrite commit history.
- Confirm the repository still has only the owner as a write or administrator collaborator.

## Public cutover

1. **Settings → General → Danger Zone → Change repository visibility → Public.**
2. Immediately create the committed rulesets:

   ```shell
   gh api --method POST repos/JamieKennedy/local-totp/rulesets --input .github/rulesets/protect-main.json
   gh api --method POST repos/JamieKennedy/local-totp/rulesets --input .github/rulesets/protect-release-tags.json
   ```

3. In **Settings → General → Pull Requests**, enable squash merging only, enable automatic branch deletion, and disable auto-merge.
4. In **Settings → Actions → General**:
   - Allow GitHub-owned actions and verified creators only.
   - Require actions to be pinned to a full commit SHA.
   - Set default `GITHUB_TOKEN` permissions to read-only.
   - Do not allow Actions to create or approve pull requests.
   - Require approval for workflows from all outside collaborators.
5. In **Settings → Code security and analysis**, enable Code Security, Secret Protection, push protection, Dependabot alerts and security updates, and private vulnerability reporting. Run CodeQL once and confirm both language analyses are clean.
6. Set the repository description, homepage (`https://jamiekennedy.github.io/local-totp/`), and topics. Keep only `JamieKennedy` as a collaborator.
7. In **Settings → Pages**, choose **GitHub Actions** as the source. Restrict the `github-pages` environment deployment branch to `main`, run the Pages workflow, and confirm HTTPS enforcement.

## `Protect main` branch ruleset

Import [`.github/rulesets/protect-main.json`](../.github/rulesets/protect-main.json) as an active branch ruleset targeting `~DEFAULT_BRANCH`:

- Empty bypass list; administrators, Actions, Dependabot, bots, and GitHub Apps cannot bypass.
- Block deletion and non-fast-forward pushes; require linear history.
- Require pull requests and allow squash merge only.
- Require zero approvals, dismiss stale approvals, do not require approval of the most recent push, and do not require code-owner review.
- Require every conversation resolved and the branch up to date.
- Require: `verify / backend`, `verify / frontend`, `verify / documentation-site`, `verify / contract`, `verify / e2e`, `verify / docker`, `verify / security`, `verify / licenses`, `pr-title`, `codeql / go`, and `codeql / javascript-typescript`.

Zero required approvals prevents a solo-maintainer lockout because authors cannot approve their own pull requests. Only users with write or administrator permission can merge; repository access must remain owner-only so only `JamieKennedy` can merge.

## `Protect release tags` tag ruleset

Import [`.github/rulesets/protect-release-tags.json`](../.github/rulesets/protect-release-tags.json) as an active tag ruleset targeting `refs/tags/v*`:

- Empty bypass list.
- Restrict updates and deletion.
- Do not restrict creation, allowing the maintainer to create a new immutable release tag once.

Do not create a protected release branch or `gh-pages` branch. `release/*` branches cannot publish by themselves, and Pages deploys an artifact through the protected `github-pages` environment.

## Release publication

1. Create and push the annotated `v1.0.0` tag on the recorded `main` SHA.
2. Confirm the release workflow publishes six binary archives, checksums, an SPDX SBOM, provenance, and the multi-architecture image.
3. Change the linked GHCR package visibility to public. Package visibility is separate from repository visibility.
4. Confirm `v1.0.0`, `1.0`, `1`, and `latest` point to one digest and test an anonymous pull.
5. Complete every post-release verification in [the release runbook](RELEASING.md).
