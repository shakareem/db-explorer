package dbexplorer

import (
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

	rows, err := h.db.Query(
		fmt.Sprintf("SELECT * FROM %s;", table),
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
			values[i] = new([]byte)
		}

		err := rows.Scan(values...)
		if err != nil {
			internalError(w, err)
			return
		}

		record := make(map[string]any, len(columns))
		for i := range columns {
			raw := *values[i].(*[]byte)
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
	record := r.Context().Value(RECORD).(map[string]any)

	err := json.NewEncoder(w).Encode(
		Response{
			map[string]any{"record": record},
		},
	)
	if err != nil {
		internalError(w, err)
		return
	}
}

func (h *handler) createRow(w http.ResponseWriter, r *http.Request) {
	table := r.Context().Value(TABLE).(string)
	columns := h.tables[table]

	var requestBody map[string]any
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		StatusBadRequest(w, err)
		return
	}

	var values []any
	var placeholders []string
	var columnNames []string

	for _, col := range columns {
		if col.Extra.Valid && col.Extra.String == "auto_increment" {
			continue
		}

		val, ok := requestBody[col.Name]
		if !ok {
			val = nil
		}

		switch v := val.(type) {
		case string:
			values = append(values, v)
		case float64:
			if strings.Contains(strings.ToUpper(col.Type), "INT") {
				values = append(values, int64(v))
			} else {
				values = append(values, v)
			}
		case bool:
			values = append(values, v)
		case nil:
			values = append(values, nil)
		default:
			values = append(values, fmt.Sprintf("%v", v))
		}

		placeholders = append(placeholders, "?")
		columnNames = append(columnNames, col.Name)
	}

	result, err := h.db.Exec(
		fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s);",
			table, strings.Join(columnNames, ", "), strings.Join(placeholders, ", "),
		), values...,
	)
	if err != nil {
		internalError(w, err)
		return
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		internalError(w, err)
		return
	}

	err = json.NewEncoder(w).Encode(
		Response{
			map[string]any{"id": lastID},
		},
	)
	if err != nil {
		internalError(w, err)
		return
	}
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

func convertValue(raw []byte, sqlType string) any {
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
