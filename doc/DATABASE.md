# Database Schema & Management

## 🗄️ Database Overview

AuraMail uses **PostgreSQL** as the primary data store. This document describes the database schema, migrations, and management procedures.

---

## 📊 Database Schema

### Users Table

```sql
CREATE TABLE users (
        id SERIAL PRIMARY KEY,
        email VARCHAR(255) NOT NULL UNIQUE,
        name VARCHAR(255),
        provider VARCHAR(50),               -- e.g., "google" (nullable in current code path)
        provider_id VARCHAR(255) NOT NULL,  -- e.g., Google sub
        refresh_token TEXT,                 -- App refresh token (JWT)
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Column Descriptions:**

| Column          | Type         | Constraint       | Purpose                         |
| --------------- | ------------ | ---------------- | ------------------------------- |
| `id`            | SERIAL       | PRIMARY KEY      | Unique user identifier          |
| `email`         | VARCHAR(255) | NOT NULL, UNIQUE | User's email (unique per user)  |
| `name`          | VARCHAR(255) | -                | User's full name                |
| `provider`      | VARCHAR(50)  | (nullable)       | OAuth provider ("google")       |
| `provider_id`   | VARCHAR(255) | NOT NULL         | Provider's unique ID for user   |
| `refresh_token` | TEXT         | -                | JWT refresh token (can be NULL) |
| `created_at`    | TIMESTAMP    | DEFAULT NOW()    | Account creation time           |
| `updated_at`    | TIMESTAMP    | DEFAULT NOW()    | Last update time                |

---

## 🔄 Data Flow

### User Creation (First Login)

```
User logs in with Google
        ↓
Check if email exists in database
        ↓
Email NOT found
        ↓
INSERT INTO users (email, name, provider_id)
VALUES ('user@gmail.com', 'John Doe', '118439...')
        ↓
User record created
        ↓
Generate app refresh token
        ↓
UPDATE users SET refresh_token = '...' WHERE id = 1
        ↓
Return tokens to client
```

### User Update (Returning User)

```
User logs in with Google
        ↓
Check if email exists in database
        ↓
Email FOUND
        ↓
UPDATE users SET name = 'John Doe' WHERE email = 'user@gmail.com'
        ↓
Generate refresh token
        ↓
UPDATE users SET refresh_token = '...' WHERE id = 1
        ↓
Return tokens to client
```

### Token Refresh

```
Client sends POST /auth/refresh with refresh_token
        ↓
SELECT * FROM users WHERE refresh_token = '...'
        ↓
Found
        ↓
Generate new access token
        ↓
Return new access token
```

### Logout

```
Client sends POST /auth/logout
        ↓
UPDATE users SET refresh_token = NULL WHERE id = 1
        ↓
Refresh token deleted
        ↓
User must login again
```

---

## 🛠️ Migrations

### What are Migrations?

Migrations are versioned SQL scripts that define database schema changes. They allow:

- Tracking database changes over time
- Easy rollback to previous schema
- Consistent setup across environments
- Version control for database structure

### Migration Tool: Goose

AuraMail uses **Goose** for database migrations.

```bash
# Install goose
go install github.com/pressly/goose/v3/cmd/goose@latest

# Apply all migrations
goose -dir internal/db/migrations postgres "postgresql://user:pass@localhost/auramail" up

# Rollback one migration
goose -dir internal/db/migrations postgres "postgresql://user:pass@localhost/auramail" down

# Check migration status
goose -dir internal/db/migrations postgres "postgresql://user:pass@localhost/auramail" status
```

### Creating New Migrations

```bash
# Create a new migration
goose -dir internal/db/migrations create create_emails sql

# A new file is created: internal/db/migrations/00002_create_emails.sql
# Edit the file to add your SQL
```

### Migration File Structure

```sql
-- +goose Up
-- SQL code to apply this migration
CREATE TABLE emails (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    subject VARCHAR(255),
    body TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- +goose Down
-- SQL code to rollback this migration
DROP TABLE emails;
```

---

## 📋 Example Queries

### View All Users

```sql
SELECT id, email, name, provider, created_at FROM users;
```

### Find User by Email

```sql
SELECT * FROM users WHERE email = 'user@gmail.com';
```

### Find User by Refresh Token

```sql
SELECT * FROM users WHERE refresh_token = 'eyJhbGc...';
```

### Count Users by Provider

```sql
SELECT provider, COUNT(*) FROM users GROUP BY provider;
```

### Find Users Without Refresh Token

```sql
SELECT * FROM users WHERE refresh_token IS NULL;
```

### Clear All Refresh Tokens (Logout Everyone)

```bash
# ⚠️ WARNING: This logs out all users!
UPDATE users SET refresh_token = NULL;
```

### Delete a User

```bash
# ⚠️ WARNING: This deletes user permanently!
DELETE FROM users WHERE id = 1;
```

---

## 🔐 Database Security

### Production Best Practices

1. **Use Environment Variables**

   ```bash
   # Never hardcode credentials
   DATABASE_URL=postgresql://user:strong_password@prod-host/auramail
   ```

2. **Use Strong Passwords**

   ```bash
   # Generate strong password
   openssl rand -base64 32
   ```

3. **Restrict Database Access**

   ```sql
   -- Create specific user for application
   CREATE USER auramail_app WITH PASSWORD 'strong_password';
   GRANT CONNECT ON DATABASE auramail TO auramail_app;
   GRANT USAGE ON SCHEMA public TO auramail_app;
   GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO auramail_app;
   ```

4. **Enable SSL for Database Connection**

   ```bash
   DATABASE_URL=postgresql://user:pass@host:5432/db?sslmode=require
   ```

5. **Regular Backups**

   ```bash
   # Daily backup
   pg_dump postgresql://user:pass@host/db > backup_$(date +%Y%m%d).sql
   ```

6. **Monitor Access**
   ```sql
   -- Enable query logging
   ALTER SYSTEM SET log_statement = 'all';
   SELECT pg_reload_conf();
   ```

---

## 🔍 Database Inspection

### Connect to Database

```bash
# Using psql CLI
psql postgresql://postgres:postgres@localhost:5432/auramail

# Or with separate params
psql -h localhost -U postgres -d auramail
```

### Common psql Commands

```sql
-- List all tables
\dt

-- Describe a table
\d users

-- Show current database
\c

-- Exit psql
\q

-- Execute SQL from file
\i migration.sql

-- Show table indexes
\di

-- Show table constraints
\d+ users
```

### Check Database Size

```sql
-- Size of entire database
SELECT pg_size_pretty(pg_database_size('auramail'));

-- Size of users table
SELECT pg_size_pretty(pg_total_relation_size('users'));

-- Size of each table
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

---

## 🔧 Maintenance Tasks

### Vacuum & Analyze

PostgreSQL maintenance to clean up space and optimize queries:

```sql
-- Vacuum (cleanup)
VACUUM ANALYZE;

-- On specific table
VACUUM ANALYZE users;
```

### Reindex

Rebuild indexes for optimal performance:

```sql
-- Reindex database
REINDEX DATABASE auramail;

-- Reindex table
REINDEX TABLE users;
```

### Check Table Integrity

```sql
-- Analyze table
ANALYZE users;

-- Get table statistics
SELECT schemaname, tablename, n_live_tup, n_dead_tup
FROM pg_stat_user_tables;
```

---

## 📈 Performance Optimization

### Add Indexes

Improve query performance for frequently filtered columns:

```sql
-- Index on email (already exists as UNIQUE)
CREATE INDEX idx_users_email ON users(email);

-- Index on provider
CREATE INDEX idx_users_provider ON users(provider);

-- Index on refresh_token
CREATE INDEX idx_users_refresh_token ON users(refresh_token);

-- Composite index
CREATE INDEX idx_users_provider_id ON users(provider, provider_id);
```

### View Index Usage

```sql
-- Unused indexes
SELECT schemaname, tablename, indexname
FROM pg_indexes
WHERE schemaname NOT IN ('pg_catalog', 'information_schema');

-- Index size
SELECT
    indexname,
    pg_size_pretty(pg_relation_size(indexrelname)) AS size
FROM pg_stat_user_indexes;
```

---

## 🐛 Common Issues

### Issue: `password authentication failed`

**Cause:** Wrong database credentials

**Solution:**

```bash
# Check .env DATABASE_URL
cat .env | grep DATABASE_URL

# Verify credentials
psql -h localhost -U postgres -d postgres -c "SELECT 1;"

# Reset password
ALTER USER postgres WITH PASSWORD 'new_password';
```

### Issue: `database "auramail" does not exist`

**Cause:** Database not created

**Solution:**

```bash
# Create database
createdb auramail

# Or via psql
psql -U postgres -c "CREATE DATABASE auramail;"
```

### Issue: `relation "users" does not exist`

**Cause:** Migrations not applied

**Solution:**

```bash
# Apply migrations
goose -dir internal/db/migrations postgres $GOOSE_DBSTRING up

# Check status
goose -dir internal/db/migrations postgres $GOOSE_DBSTRING status
```

### Issue: Slow Queries

**Solution:**

```sql
-- Enable query logging
ALTER SYSTEM SET log_min_duration_statement = 1000;  -- Log queries > 1s
SELECT pg_reload_conf();

-- Check slow queries in log
tail -f /var/log/postgresql/postgresql.log | grep "duration:"

-- Add indexes (see Performance section above)
```

---

## 📊 Monitoring

### Connection Count

```sql
SELECT count(*) FROM pg_stat_activity;

-- By user
SELECT usename, count(*) FROM pg_stat_activity GROUP BY usename;

-- By database
SELECT datname, count(*) FROM pg_stat_activity GROUP BY datname;
```

### Active Queries

```sql
SELECT pid, usename, query, state FROM pg_stat_activity WHERE state != 'idle';
```

### Kill Long-Running Query

```sql
-- Find query PID
SELECT pid, usename, query FROM pg_stat_activity;

-- Kill it
SELECT pg_terminate_backend(pid);
```

---

## 🔄 Backup & Restore

### Full Database Backup

```bash
# Create backup
pg_dump postgresql://user:pass@localhost/auramail > backup.sql

# Or with custom format (compressed)
pg_dump -Fc postgresql://user:pass@localhost/auramail > backup.dump
```

### Restore from Backup

```bash
# From SQL file
psql postgresql://user:pass@localhost/auramail < backup.sql

# From custom format
pg_restore -d auramail backup.dump
```

### Scheduled Backups (Cron)

```bash
# Add to crontab
0 2 * * * pg_dump postgresql://user:pass@localhost/auramail > /backups/auramail_$(date +\%Y\%m\%d).sql

# Or using backup tool
0 2 * * * pgbackrest backup --stanza=auramail
```

---

## 📚 Entity Relationship Diagram

```
┌─────────────────────┐
│      users          │
├─────────────────────┤
│ id (PK)             │
│ email (UNIQUE)      │
│ name                │
│ provider            │
│ provider_id         │
│ refresh_token       │
│ created_at          │
│ updated_at          │
└─────────────────────┘
        │
        │ (future relationships)
        │
        ▼
    emails, labels, etc.
```

---

## 🔗 Useful Resources

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Goose Migration Tool](https://github.com/pressly/goose)
- [PostgreSQL Best Practices](https://wiki.postgresql.org/wiki/Performance_Optimization)
- [SQL Style Guide](https://www.sqlstyle.guide/)

---

_Last Updated: December 26, 2025_
