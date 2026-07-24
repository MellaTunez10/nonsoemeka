# Database Backup & Disaster Recovery

## Overview

The `db-backup` service in `docker-compose.yml` runs a `pg_dump` on startup and then
nightly at **02:00 UTC**, writing compressed `.sql.gz` files to the `postgres_backups`
named Docker volume.  Backups older than `BACKUP_RETAIN_DAYS` (default: **7 days**) are
deleted automatically.

---

## Backup file location

Inside the container: `/backups/nonsoemeka-YYYY-MM-DDTHH-MM-SS.sql.gz`

To list all backups on the host:

```bash
docker exec nonsoemeka_db_backup ls -lh /backups/
```

---

## Copy a backup off-host

```bash
# Copy the latest backup to your current directory
docker cp nonsoemeka_db_backup:/backups/$(docker exec nonsoemeka_db_backup \
  ls /backups | sort | tail -1) ./
```

Or use `rsync` / `scp` from a remote host to pull from the server.

---

## Restore procedure

> [!CAUTION]
> Restoring overwrites all existing data in the target database.
> Stop the API and frontend first to prevent writes during restore.

```bash
# 1. Stop the API and frontend (keep the DB running)
docker compose stop api frontend

# 2. Copy your backup file into the container
docker cp ./nonsoemeka-2026-07-23T02-00-00.sql.gz \
  nonsoemeka_db_backup:/tmp/restore.sql.gz

# 3. Decompress and restore
docker exec nonsoemeka_db_backup sh -c \
  'gunzip -c /tmp/restore.sql.gz | psql -U $PGUSER -d $PGDATABASE'

# 4. Restart services
docker compose start api frontend
```

---

## Manual backup (on-demand)

```bash
docker exec nonsoemeka_db_backup sh -c \
  'pg_dump | gzip > /backups/nonsoemeka-manual-$(date +%Y%m%d%H%M%S).sql.gz'
```

---

## Configuration

| Variable             | Default | Description                                    |
|----------------------|---------|------------------------------------------------|
| `BACKUP_RETAIN_DAYS` | `7`     | Backups older than this many days are deleted  |

Add `BACKUP_RETAIN_DAYS=30` to your `.env` to keep a full month's history.

---

## Off-host / cloud backup (recommended for production)

For regulated pharmacy data you should ship backups to an external store.
Options:

- **Object storage:** Use `aws s3 cp` / `rclone` in the backup script to push to S3/R2/GCS after each dump.
- **Managed Postgres:** Switch to a managed service (AWS RDS, Supabase, Neon) that provides automated point-in-time recovery (PITR) out of the box.

To add S3 offloading, extend the `run_backup` shell function in `docker-compose.yml`:

```sh
aws s3 cp "$FNAME" s3://your-bucket/backups/ --only-show-errors
```

and add `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_DEFAULT_REGION` to the
`db-backup` service environment.
