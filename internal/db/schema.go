package db

import "strings"

type ColumnMeta struct {
	Name       string `json:"name"`
	DataType   string `json:"data_type"`
	IsNullable bool   `json:"is_nullable"`
	IsPrimary  bool   `json:"is_primary"`
}

type TableSchema struct {
	TableName string       `json:"table_name"`
	Columns   []ColumnMeta `json:"columns"`
}

func LoadTableSchema(table string) (*TableSchema, error) {
	query := `
		SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_KEY
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`

	rows, err := DB.Query(query, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schema := &TableSchema{
		TableName: table,
		Columns:   []ColumnMeta{},
	}

	for rows.Next() {
		var name, dataType, isNullable, columnKey string

		if err := rows.Scan(&name, &dataType, &isNullable, &columnKey); err != nil {
			return nil, err
		}

		schema.Columns = append(schema.Columns, ColumnMeta{
			Name:       name,
			DataType:   dataType,
			IsNullable: strings.EqualFold(isNullable, "YES"),
			IsPrimary:  strings.EqualFold(columnKey, "PRI"),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return schema, nil
}

func GetRequiredColumns(schema *TableSchema) []string {
	required := []string{}
	for _, col := range schema.Columns {
		if !col.IsNullable && !col.IsPrimary {
			required = append(required, col.Name)
		}
	}
	return required
}

func IsValidColumn(schema *TableSchema, name string) bool {
	for _, col := range schema.Columns {
		if col.Name == name {
			return true
		}
	}
	return false
}
