package dbexplorer

import (
	"database/sql"
	"net/http"
)

type handler struct {
	DB     *sql.DB
	Tables []string
}

type ctxKey int

const (
	TABLE ctxKey = iota
	ROWID
)

func NewDBExplorer(db *sql.DB) (http.Handler, error) {
	// TODO: read all tables to handler
	h := handler{DB: db}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.readAllTables)
	mux.Handle(
		"GET /{table}",
		withTableAccess(http.HandlerFunc(h.readTable)),
	)
	mux.Handle(
		"GET /{table}/{rowID}",
		withTableAccess(withRowAccess(http.HandlerFunc(h.readRow))),
	)
	mux.Handle(
		"PUT /{table}",
		withTableAccess(http.HandlerFunc(h.createRow)),
	)
	mux.Handle(
		"POST /{table}/{rowID}",
		withTableAccess(withRowAccess(http.HandlerFunc(h.updateRow))),
	)
	mux.Handle(
		"DELETE /{table}/{rowID}",
		withTableAccess(withRowAccess(http.HandlerFunc(h.deleteRow))),
	)

	return mux, nil
}

func (h *handler) readAllTables(w http.ResponseWriter, r *http.Request) {

}

func (h *handler) readTable(w http.ResponseWriter, r *http.Request) {

}

func (h *handler) readRow(w http.ResponseWriter, r *http.Request) {

}

func (h *handler) createRow(w http.ResponseWriter, r *http.Request) {

}

func (h *handler) updateRow(w http.ResponseWriter, r *http.Request) {

}

func (h *handler) deleteRow(w http.ResponseWriter, r *http.Request) {

}

func withTableAccess(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	})
}

func withRowAccess(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	})
}
