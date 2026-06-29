# Lab 21 — Backup And Restore Drill

---

## Learner Orientation

### Lab Metadata

| Item | Value |
|---|---|
| Difficulty | Intermediate |
| Estimated time | 75-120 minutes |
| VMs | 2 |
| Minimum VM RAM | 2048 MB |
| SSH ports | 2234, 2235 |
| Internet required | Yes |

### Before You Start, You Should Be Able To

- Yeast installed on a Linux/KVM host
- Comfort opening a terminal and changing directories
- Ability to run `yeast up`, `yeast ssh <instance>`, and `yeast destroy`
- Basic comfort with `curl`, `systemctl`, and reading command output
- Basic shell scripting comfort

### Where Commands Run

- Run `yeast` commands from this lab folder on your laptop.
- Run Linux service commands only after you SSH into the target VM.
- When a command says "from your laptop", leave the VM shell first with `exit`.
- When a browser URL uses `localhost`, check whether Yeast already forwarded that port for you. If not, the lab will tell you when to use a manual SSH tunnel.

### Expected Checkpoints

- After `yeast up`, `yeast status` should show the expected VM or VMs as running.
- After the main setup steps, the service, tool, or workflow introduced by the lab should respond to the verification commands.
- After `bash assets/validate.sh`, the script should report all checks passed.
- After `yeast destroy`, the lab should be cleaned up before you start the next one.

### Common Mistakes To Avoid

- Running a VM command on your laptop, or a laptop command inside the VM.
- Ignoring the forwarded port shown by `yeast up` or `yeast status`, or opening a tunnel when the lab already gave you a forwarded host port.
- Skipping validation because the final page or command "looked fine".
- Forgetting to run `yeast destroy` before moving to the next lab.

---

## The Story

Your database has data. Users have created accounts, placed orders, written posts. You have backups — maybe. But do they work? Have you ever actually restored from one?

Most teams discover their backup strategy is broken during an incident, not before one. The backup file is corrupted. The restore process takes three times longer than expected. The restored database is missing two hours of data because the backup schedule was wrong. The engineer doing the restore has never done it before on this system.

This lab forces you to practice before the incident. You will set up a PostgreSQL database, create a backup script, verify the backup, restore it to a different VM, and confirm the data is intact. Then you will set up automated backups with cron.

---

## Before You Start — Understanding The Concepts

### What Is A Database Backup?

A database backup is a copy of the database's data at a point in time. It can be used to restore the database to that state after data loss, corruption, hardware failure, or accidental deletion.

There are several types:

**Logical backup** — dumps the data as SQL statements (CREATE TABLE, INSERT). Platform-independent, easy to inspect, works across Postgres versions. `pg_dump` produces logical backups.

**Physical backup** — copies the actual data files on disk. Faster for large databases, but must be restored to the same Postgres version and hardware architecture.

**WAL archiving / streaming replication** — continuous backup by streaming database write-ahead logs to a secondary. Enables point-in-time recovery (PITR).

For this lab we use logical backups with `pg_dump` — the most common approach for small to medium databases.

### What Is `pg_dump`?

`pg_dump` is PostgreSQL's built-in backup tool. It exports a database to a SQL file that can be used to recreate it.

```bash
pg_dump -U postgres appdb > backup.sql
```

The output file contains all the SQL needed to recreate the schema and data. You restore it with `psql`:

```bash
psql -U postgres -d appdb_restored < backup.sql
```

`pg_dump` supports compressed output (`-Fc` format) for smaller files, and parallel dumps for large databases.

### What Is RPO And RTO?

**RPO (Recovery Point Objective)** — how much data can you afford to lose? If you back up every hour and the database crashes, you lose up to 1 hour of data. Your RPO is 1 hour.

**RTO (Recovery Time Objective)** — how long can recovery take? If the restore process takes 4 hours, your RTO is 4 hours. During that 4 hours, the service is down.

Both are business decisions. A financial system might require RPO of 0 (no data loss) and RTO of minutes. A blog might accept RPO of 24 hours and RTO of several hours.

Your backup strategy must satisfy your RPO and RTO. This lab teaches you to measure both on a real system.

### What Is `pg_restore`?

`pg_restore` restores backups created with `pg_dump -Fc` (the custom binary format). It supports parallel restore (`-j 4` uses 4 threads), selective restore (only specific tables), and listing the backup contents without restoring.

For plain SQL format (`pg_dump` without `-Fc`), you use `psql` directly to restore.

---

## What You Are Building

Two VMs: `primary` (the database server) and `backup` (where we test restores).

```
SSH 2234 → primary port 22  (PostgreSQL + data + backup script)
SSH 2235 → backup port 22   (restore target)

primary:
  PostgreSQL running
  appdb with sample data
  backup.sh script
  cron job for automated backups
  /home/ubuntu/backups/ directory

backup:
  PostgreSQL client tools
  restore target for drill
```

---

## Starting The Lab

```bash
cd 21-backup-restore-drill
yeast up
```

---

## Step 1 — Create The Database With Sample Data

```bash
yeast ssh primary

sudo -u postgres psql << 'SQL'
CREATE USER appuser WITH PASSWORD 'backuplab21';
CREATE DATABASE appdb OWNER appuser;
GRANT ALL PRIVILEGES ON DATABASE appdb TO appuser;
\q
SQL

sudo -u postgres psql -d appdb << 'SQL'
CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    value NUMERIC,
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO items (name, value) VALUES
    ('Widget A', 19.99),
    ('Widget B', 34.50),
    ('Service Pack', 99.00),
    ('Annual License', 499.00),
    ('Support Contract', 1200.00);

SELECT COUNT(*) AS rows, SUM(value) AS total FROM items;
SQL

exit
```

---

## Step 2 — Write The Backup Script

```bash
yeast ssh primary

mkdir -p /home/ubuntu/backups

cat > /home/ubuntu/backup.sh << 'EOF'
#!/usr/bin/env bash
# backup.sh — PostgreSQL logical backup script
set -euo pipefail

BACKUP_DIR="/home/ubuntu/backups"
DB_NAME="appdb"
DB_USER="appuser"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="${BACKUP_DIR}/${DB_NAME}_${TIMESTAMP}.sql.gz"
KEEP_DAYS=7

mkdir -p "$BACKUP_DIR"

echo "[$(date)] Starting backup of ${DB_NAME}"

# Dump the database and compress in one pipeline
PGPASSWORD="backuplab21" pg_dump \
  -U "$DB_USER" \
  -h localhost \
  "$DB_NAME" \
  | gzip > "$BACKUP_FILE"

SIZE=$(du -sh "$BACKUP_FILE" | cut -f1)
echo "[$(date)] Backup complete: ${BACKUP_FILE} (${SIZE})"

# Verify the backup is readable
if zcat "$BACKUP_FILE" | head -5 | grep -q "PostgreSQL"; then
    echo "[$(date)] Backup verification: PASSED"
else
    echo "[$(date)] Backup verification: FAILED - file may be corrupt"
    exit 1
fi

# Remove backups older than KEEP_DAYS
find "$BACKUP_DIR" -name "*.sql.gz" -mtime "+${KEEP_DAYS}" -delete
echo "[$(date)] Old backups pruned (keeping last ${KEEP_DAYS} days)"
echo "[$(date)] Done. Backups in ${BACKUP_DIR}:"
ls -lh "$BACKUP_DIR"
EOF

chmod +x /home/ubuntu/backup.sh

# Run it now
bash /home/ubuntu/backup.sh
exit
```

---

## Step 3 — Set Up PostgreSQL To Accept Local Connections

The backup script connects as `appuser` via TCP. Configure PostgreSQL to allow this:

```bash
yeast ssh primary

PG_HBA=$(sudo -u postgres psql -t -c "SHOW hba_file;" | tr -d ' ')
echo "pg_hba.conf is at: $PG_HBA"

sudo tee -a "$PG_HBA" << 'EOF'
host    appdb    appuser    127.0.0.1/32    md5
EOF

sudo systemctl reload postgresql
exit
```

Rerun the backup:

```bash
yeast ssh primary
bash /home/ubuntu/backup.sh
ls /home/ubuntu/backups/
exit
```

---

## Step 4 — Automate With Cron

```bash
yeast ssh primary

# Add to crontab: run backup daily at 2 AM
(crontab -l 2>/dev/null; echo "0 2 * * * /home/ubuntu/backup.sh >> /home/ubuntu/backup.log 2>&1") | crontab -

# Verify
crontab -l
exit
```

---

## Step 5 — Copy The Backup To The Restore Target

Simulate transferring the backup to the backup VM. In production this would go to object storage (S3, GCS) or a separate server via rsync or sftp.

```bash
# From your laptop — get the backup file from primary
BACKUP_FILE=$(ssh -p 2234 -o StrictHostKeyChecking=no ubuntu@127.0.0.1 \
  "ls /home/ubuntu/backups/*.sql.gz | head -1")
echo "Backup file: $BACKUP_FILE"

# Copy to backup VM (via your laptop as intermediary)
ssh -p 2234 -o StrictHostKeyChecking=no ubuntu@127.0.0.1 "cat $BACKUP_FILE" | \
  ssh -p 2235 -o StrictHostKeyChecking=no ubuntu@127.0.0.1 \
  "cat > /home/ubuntu/restore.sql.gz"
```

---

## Step 6 — Restore And Validate

This is the most important step. The backup only has value if the restore works.

```bash
yeast ssh backup

# Verify the file is readable
zcat /home/ubuntu/restore.sql.gz | head -10

# Install PostgreSQL server to accept the restore
sudo apt-get install -y postgresql
sudo systemctl start postgresql

# Create the restore target database
sudo -u postgres psql << 'SQL'
CREATE DATABASE appdb_restored;
\q
SQL

# Restore the backup
zcat /home/ubuntu/restore.sql.gz | \
  sudo -u postgres psql appdb_restored

# Verify the data
sudo -u postgres psql -d appdb_restored << 'SQL'
SELECT COUNT(*) AS rows FROM items;
SELECT name, value FROM items ORDER BY id;
SQL
```

Expected: 5 rows, same names and values as the original database.

**This is the key moment.** You just proved your backup works. Write down:
- How long the restore took (measure with `time`)
- What version of Postgres was used
- Whether all data was present

This is your RTO measurement.

```bash
# Time the restore (for a larger database this matters a lot)
time (zcat /home/ubuntu/restore.sql.gz | sudo -u postgres psql appdb_new 2>/dev/null || true)
exit
```

---

## Step 7 — Calculate RPO

Look at the backup script. It runs at 2 AM. If the database crashes at 1:59 AM, you lose almost 24 hours of data.

For the data in this lab, 24-hour RPO is probably fine. For a production e-commerce database taking thousands of orders per hour, it is not.

Options to reduce RPO:
- **More frequent backups** — run `backup.sh` every hour instead of every day
- **WAL archiving** — stream write-ahead logs continuously for point-in-time recovery
- **Streaming replication** — a hot standby that is always current

For now, increase the backup frequency to every hour:

```bash
yeast ssh primary
# Change cron from daily to hourly
(crontab -l | grep -v "backup.sh"; echo "0 * * * * /home/ubuntu/backup.sh >> /home/ubuntu/backup.log 2>&1") | crontab -
crontab -l
exit
```

---

## Validate Your Work

```bash
bash assets/validate.sh
```

---

## Clean Up

```bash
yeast destroy
```

---

## Quick Recap

In Lab 21 — Backup And Restore Drill, you moved from explanation to a working lab environment, verified the result, and practiced the operational habit that matters most: do the work, prove it works, then clean it up.

Keep this pattern for every lab:

1. Build the thing.
2. Verify it from the right place.
3. Read the logs or status when it fails.
4. Run the validation script.
5. Destroy the lab before moving on.

---

## What You Learned

- `pg_dump` and `pg_restore`: how PostgreSQL logical backups work
- Backup script structure: dump → compress → verify → prune old backups
- Why verification matters: a backup you have never restored is an untested assumption
- RPO: how much data you can afford to lose, and how backup frequency maps to it
- RTO: how long recovery takes, measured by actually doing it
- Cron: scheduling automated backups
- The principle: test your restore, not just your backup

---

## What Is Next

**Lab 22 — Chaos And Failure Recovery Drill**

Backups are one recovery path. Lab 22 practices broader failure recovery: taking down services one by one and recovering the platform, guided by a runbook you write before the failures start.
