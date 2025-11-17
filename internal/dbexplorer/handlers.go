package dbexplorer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Response struct {
	Response map[string]any `json:"response"`
}

func (h *handler) readAllTables(w http.ResponseWriter, r *http.Request) {
	tables := make([]string, 0, len(h.tables))
	for table := range h.tables {
		tables = append(tables, table)
	}

	err := json.NewEncoder(w).Encode(
		Response{
			map[string]any{"tables": tables},
		},
	)
	if err != nil {
		internalError(w, err)
	}
}

func (h *handler) readTable(w http.ResponseWriter, r *http.Request) {
	table := r.Context().Value(TABLE).(string)

	limitString := r.FormValue("limit")
	offsetString := r.FormValue("offset")

	limit, err := strconv.Atoi(limitString)
	if err != nil {
		limit = 5
	}
	offset, err := strconv.Atoi(offsetString)
	if err != nil {
		offset = 0
	}

	columns := h.tables[table]

	columnNames := make([]string, len(h.tables[table]))
	for i, column := range columns {
		columnNames[i] = escapeIdent(column.Name)
	}

	rows, err := h.db.Query(
		fmt.Sprintf("SELECT %s FROM %s;", strings.Join(columnNames, ", "), table),
	)
	if err != nil {
		internalError(w, err)
		return
	}

	defer rows.Close()

	records := make([]map[string]any, 0, limit)
	i := -1
	for rows.Next() {
		i++
		if i < offset {
			continue
		}

		records = append(records, map[string]any{
			// TODO
		})

		if len(records) == limit {
			break
		}
	}

	if err := rows.Err(); err != nil {
		internalError(w, err)
		return
	}

	err = json.NewEncoder(w).Encode(
		Response{
			map[string]any{"records": records},
		},
	)
	if err != nil {
		internalError(w, err)
	}
}

func (h *handler) readRow(w http.ResponseWriter, r *http.Request) {
	_ = r.Context().Value(TABLE).(string)
}

func (h *handler) createRow(w http.ResponseWriter, r *http.Request) {
	_ = r.Context().Value(TABLE).(string)
}

func (h *handler) updateRow(w http.ResponseWriter, r *http.Request) {
	_ = r.Context().Value(TABLE).(string)
}

func (h *handler) deleteRow(w http.ResponseWriter, r *http.Request) {
	_ = r.Context().Value(TABLE).(string)
}

func internalError(w http.ResponseWriter, err error) {
	msg := fmt.Sprintf(`{"error":"%s"}`, err.Error())
	http.Error(w, msg, http.StatusInternalServerError)
}

func StatusBadRequest(w http.ResponseWriter, err error) {
	msg := fmt.Sprintf(`{"error":"%s"}`, err.Error())
	http.Error(w, msg, http.StatusBadRequest)
}
