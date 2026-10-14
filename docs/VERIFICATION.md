# Verification: Docker image, smoke test and clean-clone check

Output captured from real runs, so the claims in the README (image size, non-root, healthy, smoke test, `make check`
on a clean clone) can be checked without rebuilding. It is a snapshot: rerun the commands to refresh it. The first
sections cover the Docker image; the last one covers the clean-clone check (Q-05).

- **Date:** 2026-10-13 21:26 (UTC-5), i.e. 2026-10-14 02:26 UTC
- **Image:** built with `make docker-build` (spec 005 `Dockerfile`)
- **Port:** the container was started with `make docker-run PORT=18080`, because port 8080 was already in use on the
  machine that took these captures. Inside the container the server still listens on `8080`; `PORT` only changes the
  host side of the mapping (`-p 18080:8080`), as the `ports` column below shows.

## Commands

```bash
make docker-build
make docker-run PORT=18080                       # blocks; run it in one terminal
docker images sezzle-calculator:local            # in another terminal
docker inspect --format 'User={{.Config.User}}' sezzle-calculator
docker ps --filter name=sezzle-calculator --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
make smoke BASE_URL=http://localhost:18080
docker stop sezzle-calculator                    # --rm removes the container
```

## Image size

`docker images sezzle-calculator:local`:

```text
IMAGE                     ID             DISK USAGE   CONTENT SIZE   EXTRA
sezzle-calculator:local   e3cfc91ed668       16.2MB          3.7MB   U
```

The README's "about 16 MB" is the **disk usage** column (uncompressed, as stored locally). The compressed content
size is 3.7 MB.

## Non-root user

`docker inspect --format 'User={{.Config.User}}' sezzle-calculator`:

```text
User=nonroot:nonroot
```

## Health

`docker ps --filter name=sezzle-calculator`:

```text
NAMES               IMAGE                     STATUS                   PORTS
sezzle-calculator   sezzle-calculator:local   Up 9 seconds (healthy)   0.0.0.0:18080->8080/tcp, [::]:18080->8080/tcp
```

The container was already `healthy` when its uptime read 9 seconds (the capture was taken at that moment, so it may
have become healthy slightly earlier). The `Dockerfile` sets a 10 s health-check interval with a 5 s start period.

## Smoke test

`make smoke BASE_URL=http://localhost:18080` (exit code 0):

```text
./scripts/smoke.sh ${BASE_URL:-http://localhost:8080}
ok   healthz
ok   add
ok   subtract
ok   multiply
ok   divide
ok   power
ok   sqrt
ok   percentage
ok   DIVISION_BY_ZERO
ok   UNKNOWN_OPERATION
ok   INVALID_JSON
smoke: 11 checks passed against http://localhost:18080
```

## Clean-clone check (Q-05)

Spec 004 requires `make check` to be green on a clean clone after `npm ci`. This shows it, on a clone of the
repository at commit `009a7cc` ("docs: state exactly what make check runs versus what CI runs"), with no
`node_modules` and no uncommitted files.

- **Date:** 2026-10-13, started 21:58 and finished 21:58 (UTC-5)
- **Environment:** the same machine as above (Windows, Git Bash, Go and Node as in the README prerequisites)

```bash
git clone <path to the working repository> cleanclone
cd cleanclone/frontend && npm ci
cd .. && make check
```

**Result:** `npm ci` exit code 0, then `make check` exit code **0**. Key lines of the output:

```text
No fixed sleeps in tests (backend).
No fixed sleeps in tests (frontend).
Backend coverage: 97.1% (gate: 90%)
 Test Files  7 passed (7)
      Tests  154 passed (154)
Statements   : 95.39% ( 228/239 )
Branches     : 95.71% ( 201/210 )
Functions    : 95.91% ( 47/49 )
Lines        : 94.76% ( 199/210 )
```

`coverage/backend.out`, `coverage/backend.html` and `frontend/coverage/lcov.info` were generated in the clone. As the
README explains, `make check` does not run the `cmd/server` tests or the production build; CI runs those separately.
