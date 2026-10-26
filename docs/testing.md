# Testing

Three kinds of checks:

- **Go tests** against a real PostgreSQL.
- **Frontend tests** with Vitest and jsdom.
- **Static checks:** `go vet`, frontend lint, the TypeScript build, and `shellcheck` on every tracked shell script.

The pre-commit hook runs all of them. CI ([`ci.yml`](../.github/workflows/ci.yml)) runs them again on every push, plus the test of the release version rules.

## Commands

Run from the repository root unless noted. Prefix with `mise exec --`.

```sh
cd backend && mise exec -- go vet ./...
cd backend && mise exec -- go test ./... -count=1
cd backend && mise exec -- go test ./internal/timeline -run TestName -count=1   # one Go test
cd frontend && mise exec -- npm run test                                  # all frontend tests
cd frontend && mise exec -- npx vitest run src/lib/timeline.test.ts       # one frontend file
cd frontend && mise exec -- npm run lint                                  # oxlint
cd frontend && mise exec -- npm run build                                 # tsc -b, then vite build
git ls-files -z '*.sh' .githooks/pre-commit | xargs -0 mise exec -- shellcheck   # shell scripts
.github/scripts/release-version.test.sh                                  # release version rules
```

Before calling a change done, run the full set once. It is what the hook runs.

## Go: real PostgreSQL

Store tests start a PostgreSQL container with [Testcontainers](https://golang.testcontainers.org/). Docker must be running. (`backend/internal/testutil/store.go`)

1. The first `NewStore(t)` in a package starts one container, `postgres:15.3-alpine`.
2. It applies every migration once to a template database, `migrated_template`. It reads the `.sql` files in numeric order and runs only the part above `-- +goose Down`. Goose is not used.
3. Each test copies the template into its own database, and a cleanup drops it.

Each test starts with an empty, fully migrated database. Tests share no data.

Every package that uses the database needs this in `main_test.go`, so the container is shut down afterwards:

```go
func TestMain(m *testing.M) { testutil.Run(m) }
```

`timeline`, `auth`, `weather`, `ical`, `subscription`, `assistant`, and `db` have one. `db` tests are in the external `db_test` package, because `testutil` imports `db`. `quickadd` does not, because it never touches the database.

### Writing a store test

```go
store := testutil.NewStore(t)
userID, _ := store.Register(ctx, "a@example.com", "hash")
```

For handler tests, stub the user instead of making a session:

```go
handler := timeline.New(store, testutil.SignedInAs(userID))
```

- Call handlers directly with `httptest`, as `timeline/handler_test.go` does.
- For a path value such as `{eventID}`, call `req.SetPathValue("eventID", id)`. Without the mux, path values are not set.
- Test the forbidden cases too:
  - another user's ID returns `404`;
  - a store failure is not reported as `404`. `TestStoreFailuresAreNotReportedAsNotFound` guards this.

### Testing a migration

`backend/internal/testutil/migrations_test.go` applies migrations in slices, so a test can insert data before a later migration rewrites it.

### External services

- Weather tests use a fake Open-Meteo server (`httptest`) and an injectable clock (`h.now`).
- Quick add tests use a fake Anthropic server. `NewClient` takes options so tests can point it there.
- Subscription tests serve feeds from `httptest` servers on `localhost`, so they build the client with `NewClient(true)` to allow private addresses. `TestFetch` checks that `NewClient(false)` refuses them.
- No test calls a real service.

## Frontend: Vitest and Testing Library

- `npm run test` runs Vitest with jsdom (`frontend/vite.config.ts`).
- Test files sit beside the code: `X.test.ts` or `X.test.tsx`.
- Imports are explicit. There are no globals.

Three styles, in order of how often you will use them:

1. **Pure logic** (`timelineView`, `timelineRequests`, `resolveDraft`, `formValidation`). Build a small fixture and check the output. Use zoneless local date strings like `"2026-10-15T10:00:00"`, so results do not depend on the machine's time zone.
2. **Hooks with the API mocked.** Mock `lib/api`, wrap the hook in a fresh `QueryClient`, and use `renderHook`, `act`, and `waitFor`. See `features/timeline/useTimelineMutations.test.tsx`, including the rollback case.
3. **Components** with `@testing-library/react`. Query by label or role, for example `getByLabelText("Starts hour")`. Components that read preferences render through `test/renderWithPreferences.tsx`, which defaults to 24-hour time so strings do not depend on the machine's locale.

Replace browser APIs such as `Notification` with `vi.stubGlobal`.

**Not tested:** `CalendarApp`, `App`, `WeekCalendar`, `DayEntries`. There are no end-to-end tests.

## What the hook runs

`.githooks/pre-commit`, in order:

1. lint-staged (Prettier and gofmt on staged files);
2. `shellcheck` on every tracked shell script;
3. `go vet`;
4. `go test`;
5. frontend lint;
6. frontend tests;
7. frontend build.

Enable it once per clone:

```sh
git config core.hooksPath .githooks
```

`git commit --no-verify` skips all of it. Do not use it to get past a failing test.
