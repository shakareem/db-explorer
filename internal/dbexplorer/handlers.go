package dbexplorer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
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

	sort.Strings(tables)

	err := json.NewEncoder(w).Encode(
		Response{
			map[string]any{"tables": tables},
		},
	)
	if err != nil {
		internalError(w, err)
		return
	}
}

func (h *handler) readTable(w http.ResponseWriter, r *http.Request) {
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

	table := r.Context().Value(TABLE).(string)
	columns := h.tables[table]

	columnNames := make([]string, len(columns))
	for i, column := range columns {
		columnNames[i] = escapeIdent(column.Name)
	}

	rows, err := h.db.Query(
		fmt.Sprintf("SELECT %s FROM %s;", strings.Join(columnNames, ", "), escapeIdent(table)),
	)
	if err != nil {
		internalError(w, err)
		return
	}

	defer rows.Close()

	records := make([]map[string]any, 0, limit)
	for i := 0; rows.Next(); i++ {
		if i < offset {
			continue
		}

		values := make([]any, len(columns))
		for i := range values {
			values[i] = new(sql.RawBytes)
		}

		err := rows.Scan(values...)
		if err != nil {
			internalError(w, err)
			return
		}

		record := make(map[string]any, len(columns))
		for i := range columns {
			raw := *values[i].(*sql.RawBytes)
			record[columns[i].Name] = convertValue(raw, columns[i].Type)
		}

		records = append(records, record)

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
		return
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

func convertValue(raw sql.RawBytes, sqlType string) any {
	if raw == nil {
		return nil
	}

	sqlType = strings.ToUpper(sqlType)

	switch {
	case strings.Contains(sqlType, "INT"):
		if val, err := strconv.Atoi(string(raw)); err == nil {
			return val
		}
	case strings.Contains(sqlType, "FLOAT") ||
		strings.Contains(sqlType, "DOUBLE") ||
		strings.Contains(sqlType, "DECIMAL"):
		if val, err := strconv.ParseFloat(string(raw), 64); err == nil {
			return val
		}
	case strings.Contains(sqlType, "BOOL"):
		return string(raw) == "1"
	}

	return string(raw)
}
