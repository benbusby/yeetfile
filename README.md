<div align="center">
  <img width="500" style="padding-bottom: 20px;" src="https://docs.yeetfile.com/images/yeetfile-banner.png">
  <br><br>

  <p>A privacy-focused encrypted file sending service and file/password vault</p>

[![Latest Release](https://img.shields.io/github/v/release/benbusby/yeetfile)](https://github.com/benbusby/yeetfile/releases)
[![License: AGPLv3](https://img.shields.io/github/license/benbusby/yeetfile)](https://opensource.org/license/agpl-v3)

[![Tests (CLI)](https://github.com/benbusby/yeetfile/actions/workflows/go-tests.yml/badge.svg)](https://github.com/benbusby/yeetfile/actions/workflows/go-tests.yml)
[![Tests (Web)](https://github.com/benbusby/yeetfile/actions/workflows/web-tests.yml/badge.svg)](https://github.com/benbusby/yeetfile/actions/workflows/web-tests.yml)
[![Vuln Scan](https://github.com/benbusby/yeetfile/actions/workflows/vuln-scan.yml/badge.svg)](https://github.com/benbusby/yeetfile/actions/workflows/vuln-scan.yml)

<table>
  <tr>
    <td><a href="https://sr.ht/~benbusby/yeetfile">SourceHut</a></td>
    <td><a href="https://github.com/benbusby/yeetfile">GitHub</a></td>
  </tr>
</table>
</div>
<hr>

Contents
1. [About](#about)
1. [Features](#features)
    1. [YeetFile Send](#yeetfile-send)
    1. [YeetFile Vault](#yeetfile-vault)
    1. [Accounts](#accounts)
    1. [Other](#other)
1. [How It Works / Security](#how-it-works--security)
1. [Self-Hosting](#self-hosting)
    - [Access](#access)
    - [Email Registration](#email-registration)
1. [CLI Configuration](#cli-configuration)
1. [Development](#development)
    1. [Requirements](#requirements)
    1. [Setup](#setup)
    1. [Building](#building)
    1. [Environment Variables](#environment-variables)
1. [Support](#support)

## About

YeetFile is a file vault and file/text transferring service, with both a
[web](https://yeetfile.com) and [CLI
client](https://github.com/benbusby/yeetfile/releases) officially supported.
All content is encrypted locally, and the server is incapable of decrypting any
transmitted content.

## Features

### YeetFile Send

![send example](https://docs.yeetfile.com/images/send-example.png)

- Send files and text with shareable links
  - Links don't require an account to open
- Configurable upload settings
  - Expiration date/time configurable to X minutes/hours/days (max 30 days)
  - Number of downloads (max 10)
  - Optional password protection
- Free text transfers (up to 2000 characters)

___

### YeetFile Vault

![vault example](https://docs.yeetfile.com/images/vault-example.png)

- File and password storage + folder creation
- File/password/folder sharing w/ YeetFile users
  - Read/write permissions per user
- No upload size limit

___

### Accounts

- Email not required at signup
  - Account ID-only signup allowed
  - Signup not required for text-only transfers
- Options to pay for vault/send upgrades
  - Payments handled via Stripe
  - BTC and XMR supported via BTCPay
  - Not required when self-hosting
  - Ability to recycle payment ID to remove record of payment

___

### Other

- Server-specific passwords (optional for self-hosting)
- Easily self-hosted
  - Official CLI can be configured to use any server

## How It Works / Security

See: [https://docs.yeetfile.com/security](https://docs.yeetfile.com/security/)

## Self-Hosting

You can quickly create your own instance of YeetFile using `docker compose`:

`docker compose up`

This will create the Postgres db and the server running on
http://localhost:8090. You can modify the docker-compose.yml to use an external
data volume by running `docker volume create --name=yeetfile_data` and
including the following in your docker-compose.yml:

```
volumes:
  yeetfile_data:
    external: true
```

You should create your own `.env` file with whichever variables needed to customize your instance
(see: [Environment Variables](#environment-variables)).

#### Access

When self-hosting, the web interface must be accessed either from a secure context (HTTPS/TLS) or
from the same machine the service is hosted on (`localhost` or `0.0.0.0`).

If you need to access the web interface using a machine IP on your network, for example, you can
generate a cert and set the `YEETFILE_TLS_CERT` and `YEETFILE_TLS_KEY` environment variables (see
[Environment Variables](#environment-variables))

> [!NOTE]
> This does not apply to the CLI tool. You can still use all features of YeetFile from the CLI tool
> without a secure connection.

#### Email Registration

To set up email registration for your self-hosted instance, you need to define the following environment
variables:

```sh
# The email address to use for correspondence
YEETFILE_EMAIL_ADDR=...

# The host of the email address being used
YEETFILE_EMAIL_HOST=...

# The port of the email host
YEETFILE_EMAIL_PORT=...

# The SMTP login for the email address
YEETFILE_EMAIL_USER=...

# The SMTP password for the email address
YEETFILE_EMAIL_PASSWORD=...
```

## CLI Configuration

The YeetFile CLI tool can be configured using a `config.yml` file in the following path:

**Linux/macOS**: `~/.config/yeetfile/config.yml`

**Windows**: `%AppData%\yeetfile\config.yml`

When you initially launch the CLI tool, it will create this directory and add a default `config.yml`:

```yml
server: https://yeetfile.com

# Configure the default view to enter when running the "yeetfile" command with
# no arguments. Can be set to "vault", "send", or "pass" (password vault)
default_view: "vault"

# Enable debug logging to a specific file
# debug_file: "~/.config/yeetfile/debug.log"
```

You can change the `server` directive to your own instance of YeetFile.

## Development

### Requirements

- Go v1.20+
- PostgreSQL 15+
- Node.js 22.2+
  - `npm install typescript`
- A Backblaze B2 account (optional)

### Setup

1. Start PostgreSQL
2. Create a database in PostgreSQL named "yeetfile"
    1. This can be customized to your preference, just use the `YEETFILE_DB_*` environment variables outlined
       below to configure before launching the server.
3. Build YeetFile server (see next section)
4. Run YeetFile server: `./yeetfile-server`

### Building

#### Server

`make backend`

#### CLI

`make cli`

### Environment Variables

All environment variables can be defined in a file named `.env` at the root level of the repo.

#### General Environment Variables

| Name | Purpose | Default Value | Accepted Values |
| -- | -- | -- | -- |
| YEETFILE_HOST | The host for running the YeetFile server | `0.0.0.0` | |
| YEETFILE_PORT | The port for running the YeetFile server | `8090` | |
| YEETFILE_DEBUG | Enable (1) or disable (0) debug mode on the server (do not use in production) | `0` | `0` or `1` |
| YEETFILE_STORAGE | Store files in B2 or locally on the machine running the server | `b2` | `b2` or `local` |
| YEETFILE_DB_HOST | The YeetFile PostgreSQL database host | `localhost` | |
| YEETFILE_DB_PORT | The YeetFile PostgreSQL database port | `5432` | |
| YEETFILE_DB_USER | The PostgreSQL user to access the YeetFile database | `postgres` | |
| YEETFILE_DB_PASS | The password for the PostgreSQL user | None | |
| YEETFILE_DB_NAME | The name of the database that YeetFile will use | `yeetfile` | |
| YEETFILE_DEFAULT_USER_STORAGE | The default bytes of storage to assign new users | `15000000` (15MB) | `-1` for unlimited, `> 0` bytes otherwise |
| YEETFILE_DEFAULT_USER_SEND | The default bytes a user can send | `5000000` (5MB) | `-1` for unlimited, `> 0` bytes otherwise |
| YEETFILE_SERVER_SECRET | Used for encrypting password hints and 2FA recovery codes | | 32 bytes, base64 encoded |
| YEETFILE_DOMAIN | The domain that the YeetFile instance is hosted on | `http://localhost:8090` | A valid domain string beginning with `http://` or `https://` |
| YEETFILE_SESSION_AUTH_KEY | The auth key to use for user sessions | Random value | 32-byte value, base64 encoded |
| YEETFILE_SESSION_ENC_KEY | The encryption key to use for user sessions | Random value | 32-byte value, base64 encoded |
| YEETFILE_SERVER_PASSWORD | Enables password protection for user signups | None | Any string value |
| YEETFILE_MAX_NUM_USERS | Enables a maximum number of user accounts for the instance | -1 (unlimited) | Any integer value |
| YEETFILE_SERVER_SECRET | The secret value used for encrypting user password hints | | 32-byte value, base64 encoded |
| YEETFILE_CACHE_DIR | The dir to use for caching downloaded files (B2 only) | None | Any valid directory |
| YEETFILE_CACHE_MAX_SIZE | The maximum dir size the cache can fill before removing old cached files | 0 | An int value of bytes |
| YEETFILE_CACHE_MAX_FILE_SIZE | The maximum file size to cache | 0 | An int value of bytes |
| YEETFILE_TLS_KEY | The SSL key to use for connections | | The string key contents (not a file path) |
| YEETFILE_TLS_CERT | The SSL cert to use for connections | | The string cert contents (not a file path) |
| YEETFILE_LOCKDOWN | Disables anonymous (not logged in) interactions | 0 | `1` to enable lockdown, `0` to allow anonymous usage |

#### Backblaze Environment Variables

These are required to be set if you want to use Backblaze B2 to store data that has been encrypted before upload.

| Name | Purpose |
| -- | -- |
| YEETFILE_B2_BUCKET_ID | The ID of the bucket that will be used for storing uploaded content |
| YEETFILE_B2_BUCKET_KEY_ID | The ID of the key used for accessing the B2 bucket |
| YEETFILE_B2_BUCKET_KEY | The value of the key used for accessing the B2 bucket |

#### Misc Environment Variables

These can all be safely ignored when self-hosting, but are documented here
for anyone interested in hosting a public-facing instance, with or without
paid account upgrades.

| Name | Purpose |
| -- | -- |
| YEETFILE_EMAIL_ADDR | The email address to use for correspondence |
| YEETFILE_EMAIL_HOST | The host of the email address being used |
| YEETFILE_EMAIL_PORT | The port of the email host |
| YEETFILE_EMAIL_USER | The SMTP login for the email address |
| YEETFILE_EMAIL_PASSWORD | The SMTP password for the email address |
| YEETFILE_EMAIL_NO_REPLY | The no-reply email address for correspondence |
| YEETFILE_BTCPAY_WEBHOOK_SECRET | The webhook secret for the BTCPay instance |
| YEETFILE_STRIPE_KEY | The Stripe secret key |
| YEETFILE_STRIPE_WEBHOOK_SECRET | The Stripe webhook secret |
| YEETFILE_UPGRADES_JSON | A JSON array describing the available account upgrades (see shared.Upgrade struct) |

## Support

For feature requests and bugs, you can [create an issue on
GitHub](https://github.com/benbusby/yeetfile/issues), or [submit
a ticket on SourceHut (account not required)](https://todo.sr.ht/~benbusby/yeetfile).

For issues related to the official YeetFile instance, you can reach out via
email to [support@yeetfile.com](mailto:support@yeetfile.com).

For security related issues, please [email me directly](mailto:contact@benbusby.com).
