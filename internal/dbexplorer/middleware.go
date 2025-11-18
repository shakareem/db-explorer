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
		table := r.Context().Value(TABLE).(string)
		rowID := r.PathValue("rowID")

		row := h.db.QueryRow(
			fmt.Sprintf("SELECT * FROM %s WHERE id = ?", escapeIdent(table)), rowID,
		)

		columns := h.tables[table]
		values := make([]any, len(columns))
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

		record := make(map[string]any, len(columns))
		for i := range columns {
			raw := *values[i].(*[]byte)
			record[columns[i].Name] = convertValue(raw, columns[i].Type)
		}

		ctx := context.WithValue(r.Context(), RECORD, record)
		ctx = context.WithValue(ctx, ROWID, rowID)
		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}
