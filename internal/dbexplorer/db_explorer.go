package dbexplorer

import (
	"database/sql"
	"fmt"
	"net/http"
)

type handler struct {
	db     *sql.DB
	tables map[string][]column
}

type column struct {
	Name         string
	Type         string
	Nullable     string
	Key          sql.NullString
	DefaultValue sql.NullString
	Extra        sql.NullString
}

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
		h.withTableAccess(h.withRowAccess(http.HandlerFunc(h.deleteRow))),
	)

	return mux, nil
}

func newHandler(db *sql.DB) handler {
	return handler{
		db:     db,
		tables: map[string][]column{},
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
		for tableColumns.Next() {
			var c column
			if err := tableColumns.Scan(&c.Name, &c.Type, &c.Nullable, &c.Key, &c.DefaultValue, &c.Extra); err != nil {
				tableColumns.Close()
				return err
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

		h.tables[tableName] = columns
	}

	return nil
}
