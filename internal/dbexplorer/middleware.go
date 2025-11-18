package dbexplorer

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey int

const (
	TABLE ctxKey = iota
	ROWID
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
		// TODO
		handler.ServeHTTP(w, r)
	})
}

func escapeIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}
