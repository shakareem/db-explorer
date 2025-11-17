package dbexplorer

import (
	"context"
	"database/sql"
	"encoding/json"
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

type row map[string]any

type ctxKey int

const (
	TABLE ctxKey = iota
	ROWID
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
		"PUT /{table}",
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

	defer tables.Close()

	for tables.Next() {
		var tableName string
		if err := tables.Scan(&tableName); err != nil {
			return err
		}

		tableColumns, err := h.db.Query(fmt.Sprintf("SHOW COLUMNS FROM `%s`;", tableName))
		if err != nil {
			return err
		}

		defer tableColumns.Close()

		columns := []column{}
		for tableColumns.Next() {
			var c column
			if err := tableColumns.Scan(&c.Name, &c.Type, &c.Nullable, &c.Key, &c.DefaultValue, &c.Extra); err != nil {
				return err
			}

			columns = append(columns, c)
		}

		if err := tableColumns.Err(); err != nil {
			return err
		}

		h.tables[tableName] = columns
	}

	return tables.Err()
}

func (h *handler) readAllTables(w http.ResponseWriter, r *http.Request) {
	tables := make([]string, 0, len(h.tables))
	for table := range h.tables {
		tables = append(tables, table)
	}

	err := json.NewEncoder(w).Encode(
		struct {
			Tables []string `json:"tables"`
		}{
			tables,
		},
	)
	if err != nil {
		internalError(w, err)
	}
}

func (h *handler) readTable(w http.ResponseWriter, r *http.Request) {
	_ = r.Context().Value(TABLE)
}

func (h *handler) readRow(w http.ResponseWriter, r *http.Request) {
	_ = r.Context().Value(TABLE)
}

func (h *handler) createRow(w http.ResponseWriter, r *http.Request) {
	_ = r.Context().Value(TABLE)
}

func (h *handler) updateRow(w http.ResponseWriter, r *http.Request) {
	_ = r.Context().Value(TABLE)
}

func (h *handler) deleteRow(w http.ResponseWriter, r *http.Request) {
	_ = r.Context().Value(TABLE)
}

func (h *handler) withTableAccess(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tableName := r.PathValue("table")

		ok := false
		for table := range h.tables {
			if table == tableName {
				ok = true
				break
			}
		}

		if !ok {
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), TABLE, tableName)
		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *handler) withRowAccess(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	})
}

func internalError(w http.ResponseWriter, err error) {
	msg := fmt.Sprintf(`{"message":"%s"}`, err.Error())
	http.Error(w, msg, http.StatusInternalServerError)
}
