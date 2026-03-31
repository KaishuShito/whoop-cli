# WHOOP Developer App Setup

This guide walks through creating the WHOOP Developer App needed by `whoop-cli`.

## Before You Start

You need:

- an active WHOOP membership
- a WHOOP account you can sign in with
- `whoop-cli` checked out locally

## Recommended Shortcut

If you already have the CLI built or are running it with `go run`, the easiest path is:

```bash
whoop-cli setup
```

That command opens the relevant URLs, writes `.env`, and starts OAuth authorization for you.

## Manual Setup

### 1. Open the WHOOP Developer Dashboard

Visit:

- <https://developer-dashboard.whoop.com>

Sign in with the same WHOOP account that owns the data you want to access.

### 2. Create a New Application

In the dashboard, create a new app.

Recommended app name:

- `whoop-cli`

### 3. Configure the Redirect URI

Set the redirect URI to:

```text
http://localhost:8080/callback
```

This is the local callback server that `whoop-cli` starts during OAuth authorization.

### 4. Configure Scopes

Enable these scopes:

```text
read:recovery
read:sleep
read:cycles
read:workout
read:profile
read:body_measurement
offline
```

Why `offline` matters:

- it enables refresh tokens, so the CLI can refresh access automatically when tokens expire

### 5. Copy the Credentials

After creating the app, copy:

- Client ID
- Client Secret

You will paste these into `whoop-cli setup` or your `.env` file.

### 6. Create `.env`

Example:

```dotenv
WHOOP_CLIENT_ID=your-client-id
WHOOP_CLIENT_SECRET=your-client-secret
WHOOP_REDIRECT_URI=http://localhost:8080/callback
JOURNAL_DIR=/Users/you/journal
WEATHER_ENABLED=true
WEATHER_LAT=35.6503
WEATHER_LON=139.7225
AIRQUALITY_ENABLED=true
```

Notes:

- `JOURNAL_DIR` is optional unless you plan to use `fetch --write`
- `VAULT_JOURNAL_DIR` still works as a deprecated alias
- air quality uses the same `WEATHER_LAT` and `WEATHER_LON`

### 7. Run OAuth Authorization

With the config in place:

```bash
whoop-cli auth
```

This will:

1. open the WHOOP authorization page in your browser
2. ask you to approve access
3. receive the callback on `localhost:8080`
4. save `tokens.json`

### 8. Verify

Run:

```bash
whoop-cli status
whoop-cli
whoop-cli fetch --json
```

## Token Lifecycle

`whoop-cli` stores tokens in:

- `tokens.json`

Behavior:

- access tokens are refreshed automatically when the API returns `401`
- if refresh token rotation fails, rerun `whoop-cli auth`

Common recovery command:

```bash
whoop-cli auth
```

## Troubleshooting

### Redirect URI mismatch

Symptoms:

- WHOOP rejects authorization or redirects fail

Fix:

- confirm your WHOOP app uses exactly `http://localhost:8080/callback`

### Browser opened but callback never finishes

Symptoms:

- the CLI waits until timeout

Fixes:

- make sure no other process is already using port `8080`
- confirm your firewall is not blocking the local callback
- try rerunning `whoop-cli auth`

### Tokens exist but API calls fail

Try:

```bash
whoop-cli status
whoop-cli auth
```

### You only want stdout, not file writes

That's supported. Leave `JOURNAL_DIR` unset and use:

```bash
whoop-cli
whoop-cli today --json
whoop-cli fetch
```
