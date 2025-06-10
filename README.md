<div align="center">
  <img width="500" style="padding-bottom: 20px;" src="https://docs.yeetfile.com/images/yeetfile-banner.png">
  <br><br>

  <p>A privacy-focused encrypted file sending service and file/password vault.</p>

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
    - [Storage](#storage)
    - [Access](#access)
    - [Email Registration](#email-registration)
    - [Administration](#administration)
    - [Logging](#logging)
    - [Example Configurations](#example-configurations)
      - [Systemd](#systemd)
    - [Deploying with Kamal](#deploying-with-kamal)
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
client](https://github.com/benbusby/yeetfile/releases) officially supported,
and all features of the web client are available from the CLI client.

All content is encrypted locally, and the server is incapable of decrypting any
transmitted content.

In addition to having an official instance maintained at [https://yeetfile.com](
https://yeetfile.com), YeetFile is [easily self-hosted](#self-hosting) and can
be configured to store encrypted file data locally on the server, in [Backblaze B2](
https://www.backblaze.com/cloud-storage), or using any S3-compatible storage providers
(such as AWS, [Wasabi](https://wasabi.com/cloud-object-storage),
[MinIO](https://min.io), etc).

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

### Storage

Encrypted file content can be stored either locally on the machine, in Backblaze B2, or using an
S3-compatible storage solution.

To enable:

- Backblaze B2
  - Set `YEETFILE_STORAGE=b2`
  - Set all [Backblaze environment variables](#backblaze-environment-variables)
- S3
  - Set `YEETFILE_STORAGE=s3`
  - Set all [S3 environment variables](#s3-environment-variables)
- Local storage
  - Set `YEETFILE_STORAGE=local`
  - (Optional) Set [local storage environment variables](#local-storage-environment-variables)

### Access

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

# A "no-reply" email (can be something like noreply@yourdomain.com)
YEETFILE_EMAIL_NO_REPLY=...
```

### Administration

You can declare yourself as the admin of your instance by setting the
`YEETFILE_INSTANCE_ADMIN` environment variable to your YeetFile account ID or
email address.

This will allow you to manage users and their files on the instance. Note that
file names are encrypted, but you will be able to see the following metadata
for each file:

- File ID
- Last Modified
- Size
- Owner ID

### Logging

Endpoints beginning with `/api/...` should be monitored for error codes to prevent bruteforcing.

For example:

- `/login` is the endpoint for the login web page, this only loads static content
    - This will always return a `200` response, since there is nothing sensitive about loading
      the login page.
- `/api/login` is the endpoint for submitting credentials
    - This can return an error code depending on the failure (i.e. `403` for invalid credentials,
      `404` for a non-existent user, etc)

You can limit requests to all `/api` endpoints in a Nginx config, for example, with something like
this:

```nginx
limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/m;

// ...

location /api/ {
    limit_req zone=api_limit burst=20 nodelay;
    proxy_pass http://backend;
}
```

### Example Configurations

##### Systemd

```
[Unit]
Description=Yeetfile server
Wants=network.target postgresql.service
After=network.target postgresql.service

[Service]
Type=simple
Restart=always
RestartSec=5
WorkingDirectory=/opt/yeetfile
ExecStart=/opt/yeetfile/yeetfile-server

[Install]
WantedBy=multi-user.target
```

### Deploying with Kamal

[Kamal](https://kamal-deploy.org) offers zero-downtime deploys, rolling
restarts, remote builds and more.

YeetFile supports deploying with Kamal using the `deploy.sh` script and an env
file named `kamal.env`. Before running the script, create an env file with the
following environment variables (in addition to your usual `YEETFILE_*`
environment variables:

```sh
# kamal.env
YEETFILE_SERVER_IP_LIST=<comma separated IPs> # Ex: 123.45.67.89, 98.765.43.21
YEETFILE_IMAGE_NAME=<docker image name>       # Ex: benbusby/yeetfile
KAMAL_REGISTRY_SERVER=<registry URL>          # Ex: registry.digitalocean.com
KAMAL_REGISTRY_USERNAME=<registry username>   # Ex: email@account.com
KAMAL_REGISTRY_PASSWORD=<registry token>      # Ex: **********
```

Then running `./deploy.sh kamal` will build the image and deploy it to your
server. Re-running the script will update the server with an updated image,
without any downtime when restarting.

If you want different deployments for different machines, you can create other
env files named `<type>.env` (i.e. `dev.env`, `prod.env`, etc) and then run
`deploy.sh <type>`.

**Note:** This does not deploy the required PostgeSQL database. You will need
to set that up and include the credentials in your env file. If you want to
run the database on the same server as the Kamal deployment, you can spin up
postgres using a simple `docker-compose.yml` file that specifies using the
`kamal` network.

For example:

```yml
services:
  db:
    image: postgres:16-alpine
    container_name: yeetfile-db
    restart: unless-stopped
    volumes:
      - ./data:/var/lib/postgresql/data
    environment:
      - POSTGRES_HOST_AUTH_METHOD=md5
      - POSTGRES_USER=<postgres user>          # update
      - POSTGRES_PASSWORD=<postgres password>  # update
      - POSTGRES_DB=yeetfile
    expose:
      - 5432
    healthcheck:
      test: [ "CMD-SHELL", "pg_isready -U <postgres user>" ]  # update
      interval: 3s
    networks:
      - kamal

networks:
  kamal:
    external: true
```

Then you can update `YEETFILE_DB_HOST` in your deployment `<type>.env` file to
`yeetfile-db` (along with the username and password credentials) and it should
connect correctly.

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

# Configure default values when using YeetFile Send
send:
  # Maximum number of downloads for the file
  downloads: 1

  # Maximum duration of the file before it expires
  expiration_amount: 15
  expiration_units: "minutes" # Can be "minutes", "hours", or "days"

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
| YEETFILE_MAX_SEND_DOWNLOADS | Sets the maximum number of downloads for files uploaded to YeetFile Send | 10 | `-1` for unlimited, `> 0` otherwise |
| YEETFILE_MAX_SEND_EXPIRY | Sets the maximum number of days a file uploaded to YeetFile Send can exist before deletion | 30 | `-1` for unlimited, `> 0` days otherwise |
| YEETFILE_SERVER_SECRET | The secret value used for encrypting user password hints | | 32-byte value, base64 encoded |
| YEETFILE_CACHE_DIR | The dir to use for caching downloaded files (B2 only) | None | Any valid directory |
| YEETFILE_CACHE_MAX_SIZE | The maximum dir size the cache can fill before removing old cached files | 0 | An int value of bytes |
| YEETFILE_CACHE_MAX_FILE_SIZE | The maximum file size to cache | 0 | An int value of bytes |
| YEETFILE_TLS_KEY | The SSL key to use for connections | | The string key contents (not a file path) |
| YEETFILE_TLS_CERT | The SSL cert to use for connections | | The string cert contents (not a file path) |
| YEETFILE_ALLOW_INSECURE_LINKS | Allows YeetFile Send links to include the key in a URL param | 0 | `0` (disabled) or `1` (enabled) |
| YEETFILE_INSTANCE_ADMIN | The user ID or email of the user to set as admin | | A valid YeetFile email or account ID |
| YEETFILE_LIMITER_SECONDS | The number of seconds to use in rate limiting repeated requests | 30 | Any number of seconds |
| YEETFILE_LIMITER_ATTEMPTS | The number of attempts to allow before rate limiting | 6 | Any number of requests |
| YEETFILE_LOCKDOWN | Disables anonymous (not logged in) interactions | 0 | `1` to enable lockdown, `0` to allow anonymous usage |
| YEETFILE_PROFILING | Enables server profiling on http://localhost:6060 | 0 | `1` to enable, `0` to disable (default) |
| YEETFILE_ALLOW_INVITES | Allows the YeetFile instance admin to send unique invite codes to email addresses -- must also set `YEETFILE_SERVER_PASSWORD` and setup outgoing email (see [Misc Environment Variables](#misc-environment-variables)) | 0 | `1` to enable, `0` to disable (default) |
| YEETFILE_BANNER | Can be set to a string value that will appear as an info bannner for any users logged in on the web. | | Any string |

#### Backblaze Environment Variables

These are required to be set if you want to use Backblaze B2 to store data that has been encrypted before upload.

| Name | Description |
| -- | -- |
| YEETFILE_B2_BUCKET_ID | The ID of the bucket that will be used for storing uploaded content |
| YEETFILE_B2_BUCKET_KEY_ID | The ID of the key used for accessing the B2 bucket |
| YEETFILE_B2_BUCKET_KEY | The value of the key used for accessing the B2 bucket |

#### S3 Environment Variables

These are required to be set if you want to use an S3-compatible storage solution for storing encrypted data.

| Name | Description |
| -- | -- |
| YEETFILE_S3_ENDPOINT | The S3 URL (i.e. `s3.us-west-1.wasabisys.com`) |
| YEETFILE_S3_BUCKET_NAME | The name of the S3 bucket (must be created first) |
| YEETFILE_S3_REGION_NAME | The name of the bucket region (i.e. `us-west-1`) |
| YEETFILE_S3_ACCESS_KEY_ID | The ID of the bucket access key |
| YEETFILE_S3_SECRET_KEY | The secret key value for accessing the bucket |

#### Local Storage Environment Variables

These are optional, but can help configure how local storage works (if enabled).

| Name | Description | Default Value |
| -- | -- | -- |
| YEETFILE_LOCAL_STORAGE_LIMIT | The max number of bytes the local storage directory will allow | Unlimited |
| YEETFILE_LOCAL_STORAGE_PATH | The name of the S3 bucket (must be created first) | `./uploads` |

#### Misc Environment Variables

These can all be safely ignored when self-hosting, but are documented here
for anyone interested in hosting a public-facing instance, with or without
paid account upgrades.

| Name | Description |
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
