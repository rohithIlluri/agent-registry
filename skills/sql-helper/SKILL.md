---
name: sql-helper
description: "Write, optimize, and explain SQL queries. Supports PostgreSQL, MySQL, SQLite, and BigQuery dialects."
version: "1.0.0"
license: MIT
keywords: [sql, postgres, mysql, query, optimization, explain]
category: databases
allowed-tools: [Read, Bash]
user-invocable: true
---

# SQL Helper

Write, optimize, and explain SQL queries across PostgreSQL, MySQL, SQLite, and BigQuery.

## When to invoke

Invoke for any SQL task: writing queries, optimizing slow queries, explaining execution plans, generating migrations.

## Query writing guidelines

- Use explicit column lists, never `SELECT *` in production code
- Prefer CTEs (`WITH ...`) over nested subqueries for readability
- Use parameterized queries — never interpolate user input into SQL
- Include comments for complex joins or business logic

## Optimization workflow

1. Get the slow query and EXPLAIN ANALYZE output
2. Identify the bottleneck (sequential scan, missing index, bad join order)
3. Propose the fix (new index, query rewrite, materialized view)
4. Estimate the improvement

### PostgreSQL EXPLAIN
```sql
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) <query>;
```

### Index patterns
```sql
-- Single column
CREATE INDEX CONCURRENTLY idx_table_col ON table(col);
-- Composite (order matters: equality cols first, range cols last)
CREATE INDEX CONCURRENTLY idx_table_a_b ON table(a, b);
-- Partial
CREATE INDEX CONCURRENTLY idx_active_users ON users(email) WHERE deleted_at IS NULL;
-- Expression
CREATE INDEX CONCURRENTLY idx_lower_email ON users(lower(email));
```

## Safe migrations

```sql
-- Add nullable column first, backfill, then add constraint
ALTER TABLE users ADD COLUMN tier TEXT;
UPDATE users SET tier = 'free' WHERE tier IS NULL;
ALTER TABLE users ALTER COLUMN tier SET NOT NULL;
ALTER TABLE users ALTER COLUMN tier SET DEFAULT 'free';
```

## Dialect differences

| Feature        | PostgreSQL | MySQL     | SQLite   | BigQuery  |
|----------------|------------|-----------|----------|-----------|
| String concat  | `||`       | `CONCAT()`| `||`     | `||`      |
| Upsert         | ON CONFLICT| ON DUPLICATE KEY | INSERT OR REPLACE | MERGE |
| JSON           | `->`, `->>` | `->`, `->>` | `json_extract` | `JSON_VALUE` |
| Regex          | `~`        | REGEXP    | REGEXP   | REGEXP_CONTAINS |

## Constraints

- Never generate `DROP TABLE` or destructive DDL without explicit user confirmation.
- Always use `CONCURRENTLY` for index creation on production tables.
- Flag any query that reads without a WHERE clause on large tables.
