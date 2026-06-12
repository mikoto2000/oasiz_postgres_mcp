package metadata

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Repository struct {
	db  DB
	cfg Config
}

func NewRepository(db DB, cfg Config) *Repository {
	if cfg.MaxTables <= 0 {
		cfg.MaxTables = 500
	}
	if cfg.MaxColumns <= 0 {
		cfg.MaxColumns = 200
	}
	if cfg.MaxIndexes <= 0 {
		cfg.MaxIndexes = 100
	}
	return &Repository{db: db, cfg: cfg}
}

func (r *Repository) ListSchemas(ctx context.Context) (ListSchemasOutput, error) {
	rows, err := r.db.Query(ctx, `
		select n.nspname
		from pg_catalog.pg_namespace n
		where n.nspname <> 'pg_catalog'
		  and n.nspname <> 'information_schema'
		  and n.nspname not like 'pg_toast%'
		  and n.nspname not like 'pg_temp_%'
		order by n.nspname asc`)
	if err != nil {
		return ListSchemasOutput{}, err
	}
	defer rows.Close()

	var out ListSchemasOutput
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return ListSchemasOutput{}, err
		}
		if Excluded(name, r.cfg.ExcludeSchemas) {
			continue
		}
		out.Schemas = append(out.Schemas, Schema{Name: name})
	}
	return out, rows.Err()
}

func (r *Repository) ListTables(ctx context.Context, schema string, includeViews bool) (ListTablesOutput, error) {
	if ok, err := r.schemaAllowed(ctx, schema); err != nil {
		return ListTablesOutput{}, err
	} else if !ok {
		return ListTablesOutput{Schema: schema, Error: code(ErrSchemaNotFound, "schema not found or not allowed")}, nil
	}

	relkinds := []string{"r", "p"}
	if includeViews {
		relkinds = append(relkinds, "v", "m")
	}

	rows, err := r.db.Query(ctx, `
		select c.relname,
		       c.relkind::text,
		       obj_description(c.oid, 'pg_class'),
		       count(a.attnum)::int
		from pg_catalog.pg_class c
		join pg_catalog.pg_namespace n on n.oid = c.relnamespace
		left join pg_catalog.pg_attribute a
		  on a.attrelid = c.oid
		 and a.attnum > 0
		 and not a.attisdropped
		where n.nspname = $1
		  and c.relkind = any($2)
		group by c.oid
		order by c.relname asc`, schema, relkinds)
	if err != nil {
		return ListTablesOutput{}, err
	}
	defer rows.Close()

	out := ListTablesOutput{Schema: schema}
	for rows.Next() {
		var t TableSummary
		var relkind string
		if err := rows.Scan(&t.Name, &relkind, &t.Comment, &t.ColumnCount); err != nil {
			return ListTablesOutput{}, err
		}
		if Excluded(t.Name, r.cfg.ExcludeTables) {
			continue
		}
		if len(out.Tables) >= r.cfg.MaxTables {
			return ListTablesOutput{Schema: schema, Error: code(ErrResultTooLarge, "table count exceeds configured limit")}, nil
		}
		t.Type = tableType(relkind)
		out.Tables = append(out.Tables, t)
	}
	return out, rows.Err()
}

func (r *Repository) GetTableDefinition(ctx context.Context, schema, table string, includeIndexes, includeForeignKeys, includeComments bool) (TableDefinition, error) {
	if ok, err := r.schemaAllowed(ctx, schema); err != nil {
		return TableDefinition{}, err
	} else if !ok {
		return TableDefinition{Schema: schema, Table: table, Error: code(ErrSchemaNotFound, "schema not found or not allowed")}, nil
	}

	tableOID, relkind, comment, ok, err := r.tableInfo(ctx, schema, table, includeComments)
	if err != nil {
		return TableDefinition{}, err
	}
	if !ok || Excluded(table, r.cfg.ExcludeTables) {
		return TableDefinition{Schema: schema, Table: table, Error: code(ErrTableNotFound, "table not found or not allowed")}, nil
	}

	out := TableDefinition{
		Schema:    schema,
		Table:     table,
		TableType: tableType(relkind),
		Comment:   comment,
	}

	columns, err := r.columns(ctx, tableOID, includeComments)
	if err != nil {
		return TableDefinition{}, err
	}
	if len(columns) > r.cfg.MaxColumns {
		return TableDefinition{Schema: schema, Table: table, Error: code(ErrResultTooLarge, "column count exceeds configured limit")}, nil
	}
	out.Columns = columns

	pk, err := r.primaryKey(ctx, tableOID)
	if err != nil {
		return TableDefinition{}, err
	}
	out.PrimaryKey = pk

	uniques, err := r.uniqueConstraints(ctx, tableOID)
	if err != nil {
		return TableDefinition{}, err
	}
	out.UniqueConstraints = uniques

	if includeForeignKeys {
		fks, err := r.foreignKeys(ctx, tableOID)
		if err != nil {
			return TableDefinition{}, err
		}
		out.ForeignKeys = fks
	}

	if includeIndexes {
		indexes, err := r.indexes(ctx, tableOID)
		if err != nil {
			return TableDefinition{}, err
		}
		if len(indexes) > r.cfg.MaxIndexes {
			return TableDefinition{Schema: schema, Table: table, Error: code(ErrResultTooLarge, "index count exceeds configured limit")}, nil
		}
		out.Indexes = indexes
	}

	return out, nil
}

func (r *Repository) schemaAllowed(ctx context.Context, schema string) (bool, error) {
	if IsSystemSchema(schema) || Excluded(schema, r.cfg.ExcludeSchemas) {
		return false, nil
	}

	var exists bool
	err := r.db.QueryRow(ctx, `select exists(select 1 from pg_catalog.pg_namespace where nspname = $1)`, schema).Scan(&exists)
	return exists, err
}

func (r *Repository) tableInfo(ctx context.Context, schema, table string, includeComments bool) (uint32, string, *string, bool, error) {
	var oid uint32
	var relkind string
	var comment *string
	err := r.db.QueryRow(ctx, `
		select c.oid::oid, c.relkind::text,
		       case when $3 then obj_description(c.oid, 'pg_class') else null end
		from pg_catalog.pg_class c
		join pg_catalog.pg_namespace n on n.oid = c.relnamespace
		where n.nspname = $1
		  and c.relname = $2
		  and c.relkind in ('r', 'p', 'v', 'm')`, schema, table, includeComments).Scan(&oid, &relkind, &comment)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", nil, false, nil
	}
	return oid, relkind, comment, err == nil, err
}

func (r *Repository) columns(ctx context.Context, tableOID uint32, includeComments bool) ([]Column, error) {
	rows, err := r.db.Query(ctx, `
		select a.attname,
		       pg_catalog.format_type(a.atttypid, null::integer),
		       not a.attnotnull,
		       pg_catalog.pg_get_expr(ad.adbin, ad.adrelid),
		       a.attidentity <> '',
		       a.attgenerated <> '',
		       case
		         when t.typname in ('varchar', 'bpchar') and a.atttypmod > 0 then (a.atttypmod - 4)::int
		         else null
		       end,
		       case when $2 then col_description(a.attrelid, a.attnum) else null end
		from pg_catalog.pg_attribute a
		join pg_catalog.pg_type t on t.oid = a.atttypid
		left join pg_catalog.pg_attrdef ad
		  on ad.adrelid = a.attrelid
		 and ad.adnum = a.attnum
		where a.attrelid = $1
		  and a.attnum > 0
		  and not a.attisdropped
		order by a.attnum asc`, tableOID, includeComments)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var c Column
		if err := rows.Scan(&c.Name, &c.Type, &c.Nullable, &c.Default, &c.IsIdentity, &c.IsGenerated, &c.MaxLength, &c.Comment); err != nil {
			return nil, err
		}
		columns = append(columns, c)
	}
	return columns, rows.Err()
}

func (r *Repository) primaryKey(ctx context.Context, tableOID uint32) (*KeyConstraint, error) {
	keys, err := r.keyConstraints(ctx, tableOID, "p")
	if err != nil || len(keys) == 0 {
		return nil, err
	}
	return &keys[0], nil
}

func (r *Repository) uniqueConstraints(ctx context.Context, tableOID uint32) ([]KeyConstraint, error) {
	return r.keyConstraints(ctx, tableOID, "u")
}

func (r *Repository) keyConstraints(ctx context.Context, tableOID uint32, constraintType string) ([]KeyConstraint, error) {
	rows, err := r.db.Query(ctx, `
		select con.conname,
		       array_agg(att.attname order by cols.ord)::text[]
		from pg_catalog.pg_constraint con
		join unnest(con.conkey) with ordinality as cols(attnum, ord) on true
		join pg_catalog.pg_attribute att
		  on att.attrelid = con.conrelid
		 and att.attnum = cols.attnum
		where con.conrelid = $1
		  and con.contype = $2
		group by con.oid, con.conname
		order by con.conname asc`, tableOID, constraintType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []KeyConstraint
	for rows.Next() {
		var k KeyConstraint
		if err := rows.Scan(&k.Name, &k.Columns); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *Repository) foreignKeys(ctx context.Context, tableOID uint32) ([]ForeignKey, error) {
	rows, err := r.db.Query(ctx, `
		select con.conname,
		       local_cols.columns,
		       rn.nspname,
		       rc.relname,
		       foreign_cols.columns
		from pg_catalog.pg_constraint con
		join pg_catalog.pg_class rc on rc.oid = con.confrelid
		join pg_catalog.pg_namespace rn on rn.oid = rc.relnamespace
		cross join lateral (
		  select array_agg(att.attname order by cols.ord)::text[] as columns
		  from unnest(con.conkey) with ordinality as cols(attnum, ord)
		  join pg_catalog.pg_attribute att
		    on att.attrelid = con.conrelid
		   and att.attnum = cols.attnum
		) local_cols
		cross join lateral (
		  select array_agg(att.attname order by cols.ord)::text[] as columns
		  from unnest(con.confkey) with ordinality as cols(attnum, ord)
		  join pg_catalog.pg_attribute att
		    on att.attrelid = con.confrelid
		   and att.attnum = cols.attnum
		) foreign_cols
		where con.conrelid = $1
		  and con.contype = 'f'
		order by con.conname asc`, tableOID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fks []ForeignKey
	for rows.Next() {
		var fk ForeignKey
		if err := rows.Scan(&fk.Name, &fk.Columns, &fk.ReferencedSchema, &fk.ReferencedTable, &fk.ReferencedColumns); err != nil {
			return nil, err
		}
		fks = append(fks, fk)
	}
	return fks, rows.Err()
}

func (r *Repository) indexes(ctx context.Context, tableOID uint32) ([]Index, error) {
	rows, err := r.db.Query(ctx, `
		select idx.relname,
		       coalesce(array_remove(array_agg(att.attname::text order by keys.ord), null), '{}'::text[]),
		       i.indisunique,
		       i.indisprimary,
		       pg_catalog.pg_get_indexdef(idx.oid)
		from pg_catalog.pg_index i
		join pg_catalog.pg_class idx on idx.oid = i.indexrelid
		left join unnest(i.indkey) with ordinality as keys(attnum, ord) on true
		left join pg_catalog.pg_attribute att
		  on att.attrelid = i.indrelid
		 and att.attnum = keys.attnum
		where i.indrelid = $1
		group by idx.oid, idx.relname, i.indisunique, i.indisprimary
		order by idx.relname asc`, tableOID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []Index
	for rows.Next() {
		var idx Index
		if err := rows.Scan(&idx.Name, &idx.Columns, &idx.Unique, &idx.Primary, &idx.Definition); err != nil {
			return nil, err
		}
		indexes = append(indexes, idx)
	}
	return indexes, rows.Err()
}

func code(code ErrorCode, message string) *ToolError {
	return &ToolError{Code: code, Message: message}
}
