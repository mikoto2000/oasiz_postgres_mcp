package mcpserver

import (
	"context"

	"github.com/mikoto2000/oasiz_postgres_mcp/internal/metadata"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Repository interface {
	ListSchemas(ctx context.Context) (metadata.ListSchemasOutput, error)
	ListTables(ctx context.Context, schema string, includeViews bool) (metadata.ListTablesOutput, error)
	GetTableDefinition(ctx context.Context, schema, table string, includeIndexes, includeForeignKeys, includeComments bool) (metadata.TableDefinition, error)
}

type listSchemasInput struct{}

type listTablesInput struct {
	Schema       string `json:"schema" jsonschema:"PostgreSQL schema name"`
	IncludeViews bool   `json:"include_views,omitempty" jsonschema:"include views and materialized views"`
}

type getTableDefinitionInput struct {
	Schema             string `json:"schema" jsonschema:"PostgreSQL schema name"`
	Table              string `json:"table" jsonschema:"PostgreSQL table, view, or materialized view name"`
	IncludeIndexes     bool   `json:"include_indexes,omitempty" jsonschema:"include indexes"`
	IncludeForeignKeys bool   `json:"include_foreign_keys,omitempty" jsonschema:"include foreign keys"`
	IncludeComments    bool   `json:"include_comments,omitempty" jsonschema:"include table and column comments"`
}

func New(repo Repository) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "oasiz-postgres-mcp",
		Version: "v0.1.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_schemas",
		Description: "List user-defined PostgreSQL schemas. System and configured excluded schemas are omitted.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ listSchemasInput) (*mcp.CallToolResult, metadata.ListSchemasOutput, error) {
		out, err := repo.ListSchemas(ctx)
		return nil, out, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_tables",
		Description: "List tables in a user-defined PostgreSQL schema. Views are included only when requested.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listTablesInput) (*mcp.CallToolResult, metadata.ListTablesOutput, error) {
		out, err := repo.ListTables(ctx, in.Schema, in.IncludeViews)
		return toolResult(out.Error), out, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_table_definition",
		Description: "Return structured PostgreSQL table metadata. This does not return DDL or connection credentials.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in getTableDefinitionInput) (*mcp.CallToolResult, metadata.TableDefinition, error) {
		out, err := repo.GetTableDefinition(ctx, in.Schema, in.Table, in.IncludeIndexes, in.IncludeForeignKeys, in.IncludeComments)
		return toolResult(out.Error), out, err
	})

	return server
}

func toolResult(err *metadata.ToolError) *mcp.CallToolResult {
	if err == nil {
		return nil
	}
	return &mcp.CallToolResult{IsError: true}
}
