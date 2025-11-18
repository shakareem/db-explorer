package dbexplorer

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
)

type handler struct {
	db     *sql.DB
	tables map[string]table
}

type table struct {
	Name           string
	PrimaryKeyName string
	Columns        []column
}

type column struct {
	Name            string
	Type            columnType
	IsNullable      bool
	IsAutoIncrement bool
	DefaultValue    sql.NullString
}

type columnType int

const (
	TYPESTRING columnType = iota
	TYPEINT
	TYPEFLOAT
	TYPEBOOL
)

func NewDBExplorer(db *sql.DB) (http.Handler, error) {
	h := newHandler(db)
	err := h.registerTablesAndColumns()
	if err != nil {
		return http.NotFoundHandler(), err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.readAllTables)
	mux.Handle(
		"GET /{table}",
		h.withTableAccess(http.HandlerFunc(h.readTable)),
	)
	mux.Handle(
		"GET /{table}/{rowID}",
		h.withTableAccess(h.withRowAccess(http.HandlerFunc(h.readRow))),
	)
	mux.Handle(
		"PUT /{table}/",
		h.withTableAccess(http.HandlerFunc(h.createRow)),
	)
	mux.Handle(
		"POST /{table}/{rowID}",
		h.withTableAccess(h.withRowAccess(http.HandlerFunc(h.updateRow))),
	)
	mux.Handle(
		"DELETE /{table}/{rowID}",
		h.withTableAccess(http.HandlerFunc(h.deleteRow)),
	)

	return mux, nil
}

func newHandler(db *sql.DB) handler {
	return handler{
		db:     db,
		tables: map[string]table{},
	}
}

func (h *handler) registerTablesAndColumns() error {
	tables, err := h.db.Query(`SHOW TABLES;`)
	if err != nil {
		return err
	}

	tableNames := []string{}
	for tables.Next() {
		var tableName string
		if err := tables.Scan(&tableName); err != nil {
			tables.Close()
			return err
		}
		tableNames = append(tableNames, tableName)
	}

	if err := tables.Err(); err != nil {
		tables.Close()
		return err
	}

	if err := tables.Close(); err != nil {
		return err
	}

	for _, tableName := range tableNames {
		tableColumns, err := h.db.Query(
			fmt.Sprintf("SHOW COLUMNS FROM %s;", tableName),
		)
		if err != nil {
			return err
		}

		columns := []column{}
		var primaryKeyName string
		for tableColumns.Next() {
			var c column
			var cType, cNullable, cKey, cExtra string
			err := tableColumns.Scan(&c.Name, &cType, &cNullable, &cKey, &c.DefaultValue, &cExtra)
			if err != nil {
				tableColumns.Close()
				return err
			}

			c.Type = getType(cType)
			switch cNullable {
			case "YES":
				c.IsNullable = true
			case "NO":
				c.IsNullable = false
			}

			if cKey == "PRI" {
				primaryKeyName = c.Name
			}

			if cExtra == "auto_increment" {
				c.IsAutoIncrement = true
			}

			columns = append(columns, c)
		}

		if err := tableColumns.Err(); err != nil {
			tableColumns.Close()
			return err
		}

		if err := tableColumns.Close(); err != nil {
			return err
		}

		h.tables[tableName] = table{
			Name:           tableName,
			PrimaryKeyName: primaryKeyName,
			Columns:        columns,
		}
	}

	return nil
}

func getType(sqlType string) columnType {
	sqlType = strings.ToUpper(sqlType)

	switch {
	case strings.Contains(sqlType, "INT"):
		return TYPEINT
	case strings.Contains(sqlType, "FLOAT") ||
		strings.Contains(sqlType, "DOUBLE") ||
		strings.Contains(sqlType, "DECIMAL"):
		return TYPEFLOAT
	case strings.Contains(sqlType, "BOOL"):
		return TYPEBOOL
	}

	return TYPESTRING
}
