package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robert-sjoblom/pg-inventory/internal/extractor/types"
)

// Returns a list of all tables from every database.
func (s *Store) ListTables(ctx context.Context) ([]*types.TablesInfo, error) {
	databases, err := s.ListDatabases(ctx)
	if err != nil {
		return nil, err
	}

	tables, err := forEachDatabase(ctx, databases, s.listTablesForDatabase)
	if err != nil {
		return nil, err
	}

	return tables, nil
}

const queryTables = `
WITH index_column_names AS (
    SELECT
        ix.indexrelid,
        ix.indrelid,
        array_agg(a.attname ORDER BY ord.n) AS columns
    FROM pg_index ix
    CROSS JOIN LATERAL UNNEST(ix.indkey::int[]) WITH ORDINALITY AS ord(attnum, n)
    JOIN pg_attribute a ON a.attrelid = ix.indrelid AND a.attnum = ord.attnum
    GROUP BY ix.indexrelid, ix.indrelid
),
index_data AS (
    SELECT
        ix.indrelid AS table_oid,
        ix.indexrelid AS index_oid,
        jsonb_build_object(
            'name', idx.relname,
            'columns', ic.columns,
            'is_unique', ix.indisunique,
            'is_primary', ix.indisprimary,
            'is_exclusion', ix.indisexclusion,
            'is_partial', (ix.indpred IS NOT NULL),
            'is_valid', ix.indisvalid,
            'size_bytes', pg_relation_size(ix.indexrelid),
            'definition', pg_get_indexdef(ix.indexrelid)
            -- Note: Index scan statistics (idx_scan, idx_tup_read, idx_tup_fetch) are
            -- server-scoped and belong in GetIndexStats RPC, not ListTables.
        ) AS index_json
    FROM pg_index ix
    JOIN pg_class idx ON idx.oid = ix.indexrelid AND idx.relkind = 'i'
    JOIN pg_class tbl ON tbl.oid = ix.indrelid
    LEFT JOIN index_column_names ic ON ic.indexrelid = ix.indexrelid
    WHERE
        -- Exclude indexes that are inherited from a parent table (partition indexes)
        -- This prevents duplicate indexes when a partitioned table has an index
        -- that is automatically created on all partitions
        NOT EXISTS (
            SELECT 1
            FROM pg_inherits inh
            WHERE inh.inhrelid = ix.indrelid
        )
        -- Only include indexes for tables we care about (user schemas)
        AND EXISTS (
            SELECT 1
            FROM pg_namespace n
            WHERE n.oid = tbl.relnamespace
            AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
            AND n.nspname NOT LIKE 'pg_temp_%'
            AND n.nspname NOT LIKE 'pg_toast_temp_%'
        )
),
index_agg AS (
    SELECT
        table_oid,
        jsonb_agg(index_json) AS indexes
    FROM index_data
    GROUP BY table_oid
),
constraint_data AS (
    SELECT
        con.conrelid,
        jsonb_agg(
            jsonb_build_object(
                'name', con.conname,
                'type', CASE con.contype
                    WHEN 'p' THEN 'primary_key'
                    WHEN 'u' THEN 'unique'
                    WHEN 'f' THEN 'foreign_key'
                    WHEN 'c' THEN 'check'
                    WHEN 'x' THEN 'exclusion'
                END,
                'local_columns', (
                    SELECT array_agg(att.attname ORDER BY ord.n)
                    FROM unnest(con.conkey) WITH ORDINALITY ord(attnum, n)
                    JOIN pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = ord.attnum
                ),
                'foreign_table', CASE WHEN con.contype = 'f' THEN con.confrelid::regclass::text END,
                'foreign_columns', CASE WHEN con.contype = 'f' THEN (
                    SELECT array_agg(att.attname ORDER BY ord.n)
                    FROM unnest(con.confkey) WITH ORDINALITY ord(attnum, n)
                    JOIN pg_attribute att ON att.attrelid = con.confrelid AND att.attnum = ord.attnum
                ) END,
                'definition', pg_get_constraintdef(con.oid)
            )
        ) AS constraints
    FROM pg_constraint con
    WHERE con.contype IN ('p', 'u', 'f', 'c', 'x')
    GROUP BY con.conrelid
)
SELECT
    c.oid,
    c.relname::text AS name,
    n.nspname::text AS schema,
    pg_catalog.pg_get_userbyid(c.relowner)::text AS owner,
    obj_description(c.oid, 'pg_class') AS comment,
    jsonb_build_object(
        'row_estimate', c.reltuples::bigint,
        'total_size_bytes', pg_total_relation_size(c.oid),
        'heap_size_bytes', pg_relation_size(c.oid),
        'toast_size_bytes', COALESCE(pg_relation_size(c.reltoastrelid), 0)
    ) AS stats,
    (
        SELECT jsonb_agg(
            jsonb_build_object(
                'name', att.attname,
                'type', format_type(att.atttypid, att.atttypmod),
                'not_null', att.attnotnull
            )
            ORDER BY att.attnum
        )
        FROM pg_attribute att
        WHERE att.attrelid = c.oid
        AND att.attnum > 0
        AND NOT att.attisdropped
    ) AS columns,
    ia.indexes AS indexes,
    cd.constraints AS constraints
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
LEFT JOIN index_agg ia ON ia.table_oid = c.oid
LEFT JOIN constraint_data cd ON cd.conrelid = c.oid
WHERE c.relkind = 'r'  -- Only regular tables, not partitioned ('p') or other
AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
AND n.nspname NOT LIKE 'pg_temp_%'
AND n.nspname NOT LIKE 'pg_toast_temp_%'
ORDER BY n.nspname, c.relname;
`

func (s *Store) listTablesForDatabase(ctx context.Context, dbName string) ([]*types.TablesInfo, error) {
	err := s.coordinator.Acquire(ctx, dbName)
	if err != nil {
		return nil, fmt.Errorf("acquire connection to %q: %w", dbName, err)
	}
	defer s.coordinator.Release(dbName)

	pool, err := pgxpool.New(ctx, s.connStrFor(dbName))
	if err != nil {
		return nil, fmt.Errorf("connect to %q for tables: %w", dbName, err)
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, queryTables)
	if err != nil {
		return nil, fmt.Errorf("query tables in database %q: %w", dbName, err)
	}

	defer rows.Close()

	var tables []*types.Table
	for rows.Next() {
		var table types.Table
		var statsJSON []byte
		var columnsJSON []byte
		var indexesJSON []byte
		var constraintsJSON []byte

		err := rows.Scan(
			&table.Oid,
			&table.Name,
			&table.Schema,
			&table.Owner,
			&table.Comment, // comment is optional (*string in types)
			&statsJSON,
			&columnsJSON,
			&indexesJSON,
			&constraintsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("scan table row in database %q: %w", dbName, err)
		}

		// Unmarshal stats (always present)
		if err := json.Unmarshal(statsJSON, &table.Stats); err != nil {
			return nil, fmt.Errorf("unmarshal stats for table %s: %w", table.Name, err)
		}

		// Unmarshal columns (nullable - can be NULL if no columns)
		if columnsJSON != nil {
			if err := json.Unmarshal(columnsJSON, &table.TableColumns); err != nil {
				return nil, fmt.Errorf("unmarshal columns for table %s: %w", table.Name, err)
			}
		}

		// Unmarshal indexes (nullable)
		if indexesJSON != nil {
			if err := json.Unmarshal(indexesJSON, &table.TableIndexes); err != nil {
				return nil, fmt.Errorf("unmarshal indexes for table %s: %w", table.Name, err)
			}
		}

		// Unmarshal constraints (nullable)
		if constraintsJSON != nil {
			if err := json.Unmarshal(constraintsJSON, &table.TableConstraints); err != nil {
				return nil, fmt.Errorf("unmarshal constraints for table %s: %w", table.Name, err)
			}
		}

		tables = append(tables, &table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tables in database %q: %w", dbName, err)
	}
	return []*types.TablesInfo{{
		Database: dbName,
		Tables:   tables,
	}}, nil
}
