# Deployment

This guide runs the app on one AWS EC2 instance with `docker compose`, behind HTTPS on your own domain. Caddy gets and renews the certificate. The full stack is described in [getting started](getting-started.md#full-stack-in-docker).

## What you need

- A domain, or a subdomain such as `cal.example.com`, whose DNS you control.
- An AWS account.

## 1. Create the server

1. Launch an EC2 instance with Ubuntu 24.04 or Amazon Linux 2023. A `t4g.small` (arm64, 2 GB) is enough. With less memory, the frontend build can run out of memory.
2. Attach an Elastic IP, so the address survives restarts.
3. Give it a security group with these inbound rules:

   | Port    | Source       | Why                                         |
   | ------- | ------------ | ------------------------------------------- |
   | 80/tcp  | Anywhere     | HTTP, certificate checks, redirect to HTTPS |
   | 443/tcp | Anywhere     | HTTPS                                       |
   | 443/udp | Anywhere     | HTTP/3                                      |
   | 22/tcp  | Your IP only | SSH                                         |

   Never open 5432. Compose binds PostgreSQL to the instance's loopback address, so it is unreachable even if you do, but the rule documents intent.

4. Point an `A` record for your domain at the Elastic IP. Wait until `dig +short cal.example.com` returns it. Caddy cannot get a certificate before that.
5. Install Docker Engine and the Compose plugin ([Docker docs](https://docs.docker.com/engine/install/)).

## 2. Configure

Clone the repository on the server. The committed `.env` holds development values, so production values go in an untracked `.env.prod`:

```sh
git clone <repo-url> calendar && cd calendar
cat > .env.prod <<EOF
POSTGRES_DB=calendar
POSTGRES_USER=calendar
POSTGRES_PASSWORD=$(openssl rand -hex 32)
SITE_ADDRESS=cal.example.com
SESSION_COOKIE_SECURE=true
REGISTRATION_ALLOWED_EMAILS=me@example.com,partner@example.com
EOF
chmod 600 .env.prod
```

- `REGISTRATION_ALLOWED_EMAILS` lists who may create an account. Leaving it empty lets anyone on the internet register. To invite someone later, add their email and run the start command again.
- The password goes into a database URL, so keep it URL-safe. Hex is.
- PostgreSQL sets the password only when it creates the volume. Changing it later needs `ALTER USER` as well.
- For quick add and the assistant, put `ANTHROPIC_API_KEY` in `backend/.env.local`. Without it, both report that they are not configured.
- Do not set `SUBSCRIPTIONS_ALLOW_PRIVATE`. On AWS it would let a subscription URL reach the instance metadata service at `169.254.169.254`.

## 3. Start

```sh
docker compose --env-file .env.prod up -d --build
```

`--env-file` replaces `.env`, so every compose command on the server needs it. Open `https://cal.example.com`. In the browser's developer tools, the `calendar_session` cookie should show `Secure`.

## Update

```sh
git pull
docker compose --env-file .env.prod up -d --build
```

Only changed services restart. New migrations run before the API starts.

## Backups

The data lives in the `postgres_data` volume on the instance's disk. Dump it nightly and copy the dump off the instance, for example to S3:

```sh
docker compose --env-file .env.prod exec -T db \
  sh -c 'pg_dump -U "$POSTGRES_USER" -Fc "$POSTGRES_DB"' > backup-$(date +%F).dump
```

Restore into a running stack:

```sh
docker compose --env-file .env.prod exec -T db \
  sh -c 'pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean --if-exists' < backup.dump
```

EBS snapshots of the instance's volume are a second option.
