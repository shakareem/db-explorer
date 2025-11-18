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

func ErrTypeMismatch(colName string) error {
	return fmt.Errorf("field %s have invalid type", colName)
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
		badRequest(w, err)
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

		val, err = validateColumnType(col, val)
		if err != nil {
			badRequest(w, err)
			return
		}

		values = append(values, val)
		placeholders = append(placeholders, "?")
		columnNames = append(columnNames, col.Name)
	}

	result, err := h.db.Exec(
		fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
			table, strings.Join(columnNames, ","), strings.Join(placeholders, ","),
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
	table := r.Context().Value(TABLE).(string)
	rowID := r.Context().Value(ROWID).(string)
	columns := h.tables[table]

	var requestBody map[string]any
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		badRequest(w, err)
		return
	}

	var values []any
	var columnToUpdate []string

	for _, col := range columns {
		if col.Extra.Valid && col.Extra.String == "auto_increment" {
			if _, ok := requestBody[col.Name]; ok {
				badRequest(w, ErrTypeMismatch(col.Name))
				return
			}
		}

		val, ok := requestBody[col.Name]
		if !ok {
			continue
		}

		val, err = validateColumnType(col, val)
		if err != nil {
			badRequest(w, err)
			return
		}

		values = append(values, val)
		columnToUpdate = append(columnToUpdate, fmt.Sprintf("%s = ?", col.Name))
	}

	values = append(values, rowID)

	result, err := h.db.Exec(
		fmt.Sprintf("UPDATE %s SET %s WHERE id = ?;",
			table, strings.Join(columnToUpdate, ","),
		), values...,
	)
	if err != nil {
		internalError(w, err)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		internalError(w, err)
		return
	}

	err = json.NewEncoder(w).Encode(
		Response{
			map[string]any{"updated": rowsAffected},
		},
	)
	if err != nil {
		internalError(w, err)
		return
	}
}

func (h *handler) deleteRow(w http.ResponseWriter, r *http.Request) {
	_ = r.Context().Value(TABLE).(string)
	// TODO
}

func internalError(w http.ResponseWriter, err error) {
	msg := fmt.Sprintf(`{"error":"%s"}`, err.Error())
	http.Error(w, msg, http.StatusInternalServerError)
}

func badRequest(w http.ResponseWriter, err error) {
	msg := fmt.Sprintf(`{"error":"%s"}`, err.Error())
	http.Error(w, msg, http.StatusBadRequest)
}

func convertValue(raw []byte, columnType columnType) any {
	if raw == nil {
		return nil
	}

	switch columnType {
	case TYPEINT:
		if val, err := strconv.Atoi(string(raw)); err == nil {
			return val
		}
	case TYPEFLOAT:
		if val, err := strconv.ParseFloat(string(raw), 64); err == nil {
			return val
		}
	case TYPEBOOL:
		return string(raw) == "1"
	}

	return string(raw)
}

func validateColumnType(col column, val any) (any, error) {
	ErrTypeMismatch := ErrTypeMismatch(col.Name)

	switch v := val.(type) {
	case string:
		if col.Type != TYPESTRING {
			return struct{}{}, ErrTypeMismatch
		}
		return v, nil
	case float64:
		if v == float64(int64(v)) && col.Type == TYPEINT {
			return int64(v), nil
		} else {
			if col.Type != TYPEFLOAT {
				return struct{}{}, ErrTypeMismatch
			}
			return v, nil
		}
	case bool:
		if col.Type != TYPEBOOL {
			return struct{}{}, ErrTypeMismatch
		}
		return v, nil
	case nil:
		if !col.Nullable {
			return struct{}{}, ErrTypeMismatch
		}
		return nil, nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}
