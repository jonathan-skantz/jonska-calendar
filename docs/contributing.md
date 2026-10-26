# Contributing

Working rules for people. [CLAUDE.md](../CLAUDE.md) has a shorter version for AI agents. [PRODUCT.md](../PRODUCT.md) decides what to build.

## Before you change anything

1. Read the part of [PRODUCT.md](../PRODUCT.md) your change touches. **Future** items are out of scope unless the maintainer asks.
2. Read the matching parts of [architecture](architecture.md) and [data model](data-model.md).
3. For a large change, write down the design and trade-offs first. If a product rule is unclear and affects the code, ask before you write it.

## Code rules

- **Domain rules live in the backend.** The frontend is not an authorization boundary.
- **Keep owner data and share data apart.** Share fields never overwrite the event. See [data model: sharing](data-model.md#sharing).
- **Use a transaction** when one operation writes `timeline_items` and a subtype row (`events` or `tasks`), or several rows that must agree.
- **Filter every write by `user_id`** in SQL, as well as in the handler.
- **Validate input at the edge.** Handlers validate in `validation.go`. Schema constraints are the last line.
- **Keep it simple.** Follow the existing pattern. Add no abstraction or dependency without a concrete need.
- **Keep the custom week grid.** Do not swap in a calendar library unless product scope changes. See [decisions](decisions.md#the-custom-week-grid).
- **Use `date-fns`** for date math. Do not mutate `Date` objects.
- **Use TanStack Query** for server data. Do not copy it into your own state.

## Comments

- Comment non-obvious code: ownership boundaries, derived values, and SQL a newcomer would struggle to follow.
- Do not comment what the code already says.
- Use plain English and short sentences.

## Migrations

- Never edit a migration that has been applied anywhere. Add a new numbered file in `backend/db/migrations`.
- The next number is one more than the highest existing file. Numbers are not zero-padded.
- Write the `Up` part. The `Down` part is kept for reference only; tests do not run it.
- If the migration changes existing data, add a test. See [testing: migrations](testing.md#testing-a-migration).

## Tests

- Add or update a test for every behavior you change, and for the edge cases that matter.
- Run the targeted test first, then the full set in [testing](testing.md#commands).

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org). Match the style in `git log`:

```text
feat(events): add approximate start and end times
fix(validation): reject blank titles
refactor(validation): consistent validators and shared limits
test(backend): share one Postgres container per package
```

Commit only when the maintainer asks you to.

## Pre-commit hook

```sh
git config core.hooksPath .githooks
```

The hook formats staged files and runs every check. Fix the failure. Do not skip the hook with `--no-verify`.

## Branches and releases

Work on a branch and open a pull request to `main`. CI ([`ci.yml`](../.github/workflows/ci.yml)) runs the same checks on every push and builds the Docker images. PRs are squash-merged, so the PR title must be a Conventional Commit: it becomes the commit on `main` and picks the version. A merge does not deploy: the **Release** workflow tags the tip of `main` as `vX.Y.Z` and deploys it when you run it. See [deployment](deployment.md#4-release).

The workflow steps live in [`.github/scripts`](../.github/scripts), each with usage notes at the top. After changing the version rules, run `.github/scripts/release-version.test.sh`; CI runs it too, with `shellcheck`.

## Agent instructions

- [CLAUDE.md](../CLAUDE.md) links to this reading order.
- [`.github/copilot-instructions.md`](../.github/copilot-instructions.md) links only to [decisions](decisions.md).
- If you change a working rule here, check whether either agent file needs the same change.
