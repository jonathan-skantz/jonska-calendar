# Deployment

This guide runs the app with `docker compose` on one AWS EC2 instance (a virtual machine), over HTTPS on your own domain. Caddy, the web server in the `web` image, handles the certificate. The stack itself is described in [getting started](getting-started.md#full-stack-in-docker).

## How a change reaches production

| You                | What happens                                                                                             |
| ------------------ | -------------------------------------------------------------------------------------------------------- |
| Push a branch      | [CI](../.github/workflows/ci.yml) runs the checks and test-builds the images `api`, `migrate` and `web`. |
| Merge a PR to main | CI runs again and pushes the images to GHCR, tagged with the commit hash (`sha-1fc361e…`).               |
| Run **Release**    | [Release](../.github/workflows/release.yml) tags the commit and its images `vX.Y.Z` and deploys them.    |

So `main` may be ahead of production, and production runs exactly the images CI tested. Nothing is rebuilt for a release.

Terms:

- **GHCR**: GitHub Container Registry, where CI stores the images and the server pulls them from.
- **IAM role**: an AWS identity with permissions. Whoever may "assume" it gets temporary credentials.
- **OIDC**: lets a GitHub Actions job prove to AWS which repository and environment it runs in, so it can assume a role without a stored key.
- **SSM**: AWS Systems Manager. Its agent on the instance runs commands sent through AWS. Deploys use it instead of SSH.
- **GitHub environment**: named variables and rules (here `production`) for a deploying job.
- **Squash merge**: a PR merged as one commit, with the PR title as its message.

## What you need

- A domain or subdomain (e.g. `cal.example.com`) whose DNS you control.
- An AWS account.

## Once or repeated?

| Part                                                 | How often                                   |
| ---------------------------------------------------- | ------------------------------------------- |
| [1. Create the server](#1-create-the-server)         | Once per server                             |
| [2. Configure](#2-configure)                         | Once per server                             |
| [3. Connect GitHub](#3-connect-github-to-the-server) | Once (3.1 after the first CI run on `main`) |
| [4. Release](#4-release)                             | Every time you ship                         |
| [5. Roll back](#5-roll-back)                         | When a release is broken                    |
| [On the server](#on-the-server)                      | When you change a setting or debug          |
| [Backups](#backups)                                  | Nightly (automate it)                       |

## 1. Create the server

_Once per server._

1. **Launch** an EC2 instance with Ubuntu 24.04 or Amazon Linux 2023. A `t4g.small` (2 GB) is enough. It must be arm64 (Graviton), since CI builds arm64 images only.
2. **Attach an Elastic IP**, so the address survives restarts.
3. **Add a security group** (firewall) with these inbound rules:

   | Port    | Source       | Why                                         |
   | ------- | ------------ | ------------------------------------------- |
   | 80/tcp  | Anywhere     | HTTP, certificate checks, redirect to HTTPS |
   | 443/tcp | Anywhere     | HTTPS                                       |
   | 443/udp | Anywhere     | HTTP/3                                      |
   | 22/tcp  | Your IP only | SSH for the first setup; optional later     |

   Never open 5432 (PostgreSQL). Compose binds it to loopback anyway.

4. **Point DNS** at it: an `A` record to the Elastic IP. Wait until `dig +short cal.example.com` returns it; Caddy needs that to get a certificate.
5. **Install Docker** Engine and the Compose plugin ([Docker docs](https://docs.docker.com/engine/install/)). Then add the app's user to the `docker` group and log in again:

   ```sh
   sudo usermod -aG docker ubuntu   # the user is ec2-user on Amazon Linux
   ```

6. **Enable SSM**: attach an IAM instance profile with the AWS policy `AmazonSSMManagedInstanceCore`. The agent is preinstalled. Within minutes the instance should appear in Systems Manager → Fleet Manager; if not, deploys cannot reach it.

## 2. Configure

_Once per server._ You never pull or re-clone by hand: each release checks out its tag in this clone, so `compose.yaml` and `deploy.sh` always match the deployed version. `.env.prod` and `backend/.env.local` are ignored by git, so releases leave them alone.

Log in as the user from step 5 (SSH, or Session Manager in the EC2 console), clone the repository, and put the production values in `.env.prod` (the committed `.env` is for development):

```sh
git clone https://github.com/jonathan-skantz/jonska-calendar.git ~/calendar && cd ~/calendar
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

- **Keep the folder name `calendar`.** Compose names the database volume after it (`calendar_postgres_data`).
- **`REGISTRATION_ALLOWED_EMAILS`**: who may create an account. Empty means anyone. To invite someone, add the email and [apply the change](#change-a-setting).
- **The password** must be URL-safe (hex is). PostgreSQL reads it only when it creates the volume; to change it later, also run `ALTER USER`.
- **`ANTHROPIC_API_KEY`** goes in `backend/.env.local`; without it, quick add and the assistant are off. They share 200 requests a day (`LLM_DAILY_REQUEST_LIMIT` in `.env.prod`).
- **API settings** passed by compose (`SESSION_COOKIE_SECURE`, `REGISTRATION_ALLOWED_EMAILS`, `LLM_DAILY_REQUEST_LIMIT`) go in `.env.prod` and win over `backend/.env.local`.
- **Never set `SUBSCRIPTIONS_ALLOW_PRIVATE`.** On AWS it would let a subscription URL reach the instance metadata service.
- **Don't edit tracked files in the clone.** Local changes make the next release's checkout fail.

Nothing runs yet; the first release starts the stack.

## 3. Connect GitHub to the server

_Once._ Only 3.3 and 3.4 change if you move to a new server (new instance ID).

### 3.1 Make the images public

The first CI run on `main` creates three GHCR packages: `jonska-calendar-api`, `-migrate` and `-web`. They start **private**, so the server cannot pull them. For each: your GitHub profile → **Packages** → the package → **Package settings** → **Change visibility** → public.

### 3.2 Let AWS trust GitHub

IAM → Identity providers → add an OpenID Connect provider with URL `https://token.actions.githubusercontent.com` and audience `sts.amazonaws.com`.

### 3.3 Create the deploy role

IAM → Roles → create a role with the two policies below. Replace `<account>` (12-digit ID), `<region>` and `<instance-id>`.

**Trust policy**: only jobs in this repository's `production` environment may assume the role.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::<account>:oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
          "token.actions.githubusercontent.com:sub": "repo:jonathan-skantz/jonska-calendar:environment:production"
        }
      }
    }
  ]
}
```

**Permissions policy**: run a shell command on this one instance and read the result, nothing else. (`GetCommandInvocation` can't be limited to a resource, hence `*`.)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "ssm:SendCommand",
      "Resource": [
        "arn:aws:ec2:<region>:<account>:instance/<instance-id>",
        "arn:aws:ssm:<region>::document/AWS-RunShellScript"
      ]
    },
    {
      "Effect": "Allow",
      "Action": "ssm:GetCommandInvocation",
      "Resource": "*"
    }
  ]
}
```

### 3.4 Create the `production` environment

Repository **Settings** → **Environments** → **New environment** → `production`.

- **Deployment branches and tags**: allow only `main`. The role trusts this environment, so otherwise any branch could deploy.
- **Variables** (none is secret):

  | Variable          | Example                              | What it is                |
  | ----------------- | ------------------------------------ | ------------------------- |
  | `AWS_REGION`      | `eu-north-1`                         | Region of the instance    |
  | `AWS_ROLE_ARN`    | `arn:aws:iam::<account>:role/<name>` | The role from 3.3         |
  | `EC2_INSTANCE_ID` | `i-0123456789abcdef0`                | The instance              |
  | `DEPLOY_USER`     | `ubuntu`                             | Owner of the clone        |
  | `DEPLOY_DIR`      | `/home/ubuntu/calendar`              | The clone                 |
  | `SITE_URL`        | `https://cal.example.com`            | Checked after each deploy |

### 3.5 Protect `main`

**Settings** → **Branches** (or **Rules**). Require:

- a pull request, up to date with `main`;
- the checks `backend`, `frontend`, `scripts`, `images (api)`, `images (migrate)` and `images (web)`. GitHub lists a check only after it has run once.

### 3.6 Allow only squash merges

**Settings** → **General** → **Pull Requests**: allow only squash merging, with the PR title as the default message. The version is computed from these titles, so write them as [Conventional Commits](https://www.conventionalcommits.org), e.g. `feat(events): add reminders`.

Once this works, you can remove the port 22 rule: deploys don't use SSH, and Session Manager replaces it for logging in.

## 4. Release

_Every time you ship._ This is the only step a normal release needs.

Once CI on `main` has passed: **Actions** → **Release** → **Run workflow** on `main`, with `tag` empty. The workflow:

1. **Picks the version** from the PR titles since the last tag. The biggest bump wins:

   | PR title                                      | v1.2.3 becomes |
   | --------------------------------------------- | -------------- |
   | `feat(api)!: …`, or a `BREAKING CHANGE:` line | `v2.0.0`       |
   | `feat: …`                                     | `v1.3.0`       |
   | anything else (`fix:`, `docs:`, `ci:`, …)     | `v1.2.4`       |

   The first release counts from `v0.0.0` (here giving `v0.1.0`). If the commit is already tagged, it just redeploys that tag.

2. **Tags the images, then the commit.** If CI has no images for the commit yet, it stops before tagging anything.
3. **Deploys through SSM.** On the server it checks out the tag and runs [`deploy.sh`](../deploy.sh), which pulls the images and runs `docker compose up -d`. Changed containers are recreated, and `migrate` applies new migrations before `api` starts. The server output is in the workflow log.
4. **Checks the site.** `SITE_URL` must load, and `/api/auth/me` must return `401` (correct without a session, and proof that the API is up).

After the first release only, log in at `https://cal.example.com`. The `calendar_session` cookie (browser developer tools → Application → Cookies) should be marked `Secure`.

## 5. Roll back

_When a release is broken._

Run **Release** with an older tag (e.g. `v1.2.0`) in the `tag` field. The server switches to that version; nothing new is tagged. List tags with `git tag --list 'v*'`.

Migrations do not roll back, so the older API runs on the newer schema. That is fine for additive changes (a new column); after a rename or drop, restore a [backup](#backups) instead.

## On the server

_As needed: to change a setting, debug, or deploy by hand._

- Every compose command needs `--env-file .env.prod`.
- `up` and `run` also need `APP_VERSION=<deployed tag>`. Without it, compose builds `:local` images on the server instead of using the tested ones.

```sh
cd ~/calendar
git describe --tags                                          # deployed version
docker compose --env-file .env.prod ps                       # running containers
docker compose --env-file .env.prod logs -f --tail 100 api   # logs (or web, migrate, db)
```

### Change a setting

After editing `.env.prod`, run **Release** with the current tag, or:

```sh
APP_VERSION=$(git describe --tags) docker compose --env-file .env.prod up -d --no-build
```

### Deploy by hand

If the workflow can't reach the server, this does the same as its deploy step. It works only for tags that Release created, since only those images exist.

```sh
cd ~/calendar
git fetch --tags && git checkout --detach v1.3.0
./deploy.sh v1.3.0
```

### When a release fails

| Message in the Release log                                | Fix                                                             |
| --------------------------------------------------------- | --------------------------------------------------------------- |
| `No api image for <commit>`                               | CI on `main` is still running or failed. Rerun after it passes. |
| `Not authorized to perform sts:AssumeRoleWithWebIdentity` | Check the trust policy's `sub`, and that the run was on `main`. |
| `pull access denied` or `unauthorized`                    | A GHCR package is still private (3.1).                          |
| `Deploy on the instance ended with status Failed`         | Read the server output just above it.                           |
| `… is not healthy after the deploy`                       | Read `logs api` and `logs web` on the server.                   |

## Backups

_Nightly._ Set it up once, e.g. as a cron job.

The data is in the `calendar_postgres_data` volume. Dump it nightly and copy the dump off the instance (e.g. to S3):

```sh
docker compose --env-file .env.prod exec -T db \
  sh -c 'pg_dump -U "$POSTGRES_USER" -Fc "$POSTGRES_DB"' > backup-$(date +%F).dump
```

Restore into a running stack:

```sh
docker compose --env-file .env.prod exec -T db \
  sh -c 'pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean --if-exists' < backup.dump
```

EBS snapshots of the instance's disk are a second option.
