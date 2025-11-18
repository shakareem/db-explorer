package dbexplorer

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
)

type ctxKey int

const (
	TABLE ctxKey = iota
	ROWID
	RECORD
)

func (h *handler) withTableAccess(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tableName := r.PathValue("table")

		if _, ok := h.tables[tableName]; !ok {
			http.Error(w, `{"error": "unknown table"}`, http.StatusNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), TABLE, tableName)
		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *handler) withRowAccess(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tableName := r.Context().Value(TABLE).(string)
		rowID := r.PathValue("rowID")
		table := h.tables[tableName]

		row := h.db.QueryRow(
			fmt.Sprintf("SELECT * FROM %s WHERE %s = ?;",
				tableName, table.PrimaryKeyName), rowID,
		)

		values := make([]any, len(table.Columns))
		for i := range values {
			values[i] = new([]byte)
		}

		err := row.Scan(values...)
		if err == sql.ErrNoRows {
			http.Error(w, `{"error": "record not found"}`, http.StatusNotFound)
			return
		}
		if err != nil {
			internalError(w, err)
			return
		}
		if row.Err() != nil {
			internalError(w, err)
			return
		}

		record := make(map[string]any, len(table.Columns))
		for i := range table.Columns {
			raw := *values[i].(*[]byte)
			record[table.Columns[i].Name] = convertValue(raw, table.Columns[i].Type)
		}

		ctx := context.WithValue(r.Context(), RECORD, record)
		ctx = context.WithValue(ctx, ROWID, rowID)
		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}
