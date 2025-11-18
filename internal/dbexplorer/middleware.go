package dbexplorer

import (
	"context"
	"fmt"
	"net/http"
)

type ctxKey int

const (
	TABLE ctxKey = iota
	ROWID
	ROW
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
		if row.Err() != nil {
			http.Error(w, `{"error": "record not found"}`, http.StatusNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), ROW, row)
		ctx = context.WithValue(ctx, ROWID, rowID)
		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}
