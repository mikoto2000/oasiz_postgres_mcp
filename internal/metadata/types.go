package metadata

type ErrorCode string

const (
	ErrSchemaNotFound ErrorCode = "SCHEMA_NOT_FOUND"
	ErrTableNotFound  ErrorCode = "TABLE_NOT_FOUND"
	ErrResultTooLarge ErrorCode = "RESULT_TOO_LARGE"
)

type ToolError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type Config struct {
	ExcludeSchemas []string `json:"exclude_schemas"`
	ExcludeTables  []string `json:"exclude_tables"`
	MaxTables      int      `json:"max_tables"`
	MaxColumns     int      `json:"max_columns"`
	MaxIndexes     int      `json:"max_indexes"`
}

type Schema struct {
	Name string `json:"name"`
}

type ListSchemasOutput struct {
	Schemas []Schema   `json:"schemas,omitempty"`
	Error   *ToolError `json:"error,omitempty"`
}

type TableSummary struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Comment     *string `json:"comment"`
	ColumnCount int     `json:"column_count"`
}

type ListTablesOutput struct {
	Schema string         `json:"schema,omitempty"`
	Tables []TableSummary `json:"tables,omitempty"`
	Error  *ToolError     `json:"error,omitempty"`
}

type Column struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Nullable    bool    `json:"nullable"`
	Default     *string `json:"default"`
	IsIdentity  bool    `json:"is_identity"`
	IsGenerated bool    `json:"is_generated"`
	MaxLength   *int32  `json:"max_length,omitempty"`
	Comment     *string `json:"comment,omitempty"`
}

type KeyConstraint struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
}

type ForeignKey struct {
	Name              string   `json:"name"`
	Columns           []string `json:"columns"`
	ReferencedSchema  string   `json:"referenced_schema"`
	ReferencedTable   string   `json:"referenced_table"`
	ReferencedColumns []string `json:"referenced_columns"`
}

type Index struct {
	Name       string   `json:"name"`
	Columns    []string `json:"columns,omitempty"`
	Unique     bool     `json:"unique"`
	Primary    bool     `json:"primary"`
	Definition string   `json:"definition"`
}

type TableDefinition struct {
	Schema            string          `json:"schema,omitempty"`
	Table             string          `json:"table,omitempty"`
	TableType         string          `json:"table_type,omitempty"`
	Comment           *string         `json:"comment,omitempty"`
	Columns           []Column        `json:"columns,omitempty"`
	PrimaryKey        *KeyConstraint  `json:"primary_key,omitempty"`
	ForeignKeys       []ForeignKey    `json:"foreign_keys,omitempty"`
	UniqueConstraints []KeyConstraint `json:"unique_constraints,omitempty"`
	Indexes           []Index         `json:"indexes,omitempty"`
	Error             *ToolError      `json:"error,omitempty"`
}
