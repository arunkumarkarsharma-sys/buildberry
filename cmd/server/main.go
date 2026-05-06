package main

import (
	"buildberry/internal/config"
	db "buildberry/internal/db"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

func main() {
	cfg := config.LoadConfig()

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	http.HandleFunc("/api/import-excel/", importExcelHandler)

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "pong")
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok", "service":"buildberry"}
		`))

	})
	http.HandleFunc("/api/tables", func(w http.ResponseWriter, r *http.Request) {
		//QUERY RUN

		rows, err := database.Query("SHOW TABLES")
		if err != nil {
			http.Error(w, err.Error(), 500)

			return
		}
		defer rows.Close()

		//slice making

		tables := []string{}
		fmt.Println("query executed")

		//loop and scan

		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			tables = append(tables, table)
		}
		//json

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tables": tables,
		})

	})

	//route adding
	http.HandleFunc("/api/schema/", func(w http.ResponseWriter, r *http.Request) {

		//table name find out

		table := strings.TrimPrefix(r.URL.Path, "/api/schema/")

		//Query run
		rows, err := database.Query("DESCRIBE " + table)
		if err != nil {
			http.Error(w, err.Error(), 500)

			return
		}
		defer rows.Close()
		//struct
		type Column struct {
			Name string `json:"name"`

			Type string `json:"type"`
		}

		//loop and scan

		columns := []Column{}

		for rows.Next() {
			var field, colType, null, key, extra string
			var defaultVal *string

			rows.Scan(&field, &colType, &null, &key, &defaultVal, &extra)

			columns = append(columns, Column{
				Name: field,
				Type: colType,
			})
		}

		//json responce
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"columns": columns,
		})

	})

	http.HandleFunc("/api/data/", func(w http.ResponseWriter, r *http.Request) {
		table := strings.TrimPrefix(r.URL.Path, "/api/data/")

		rows, err := database.Query("SELECT * FROM " + table)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()

		columns, _ := rows.Columns()
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		var results []map[string]interface{}

		for rows.Next() {
			for i := range columns {
				valuePtrs[i] = &values[i]
			}

			rows.Scan(valuePtrs...)

			row := make(map[string]interface{})
			for i, col := range columns {
				val := values[i]
				b, ok := val.([]byte)
				if ok {
					row[col] = string(b)
				} else {
					row[col] = val
				}
			}

			results = append(results, row)
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": results,
		})

	})
	//route add(csv export)
	http.HandleFunc("/api/export/", func(w http.ResponseWriter, r *http.Request) {

		//table extract
		table := strings.TrimPrefix(r.URL.Path, "/api/export/")
		table = strings.TrimSuffix(table, ".csv")

		//whitelist
		allowedTables := map[string]bool{
			"users": true,
		}

		if !allowedTables[table] {
			http.Error(w, "invalid table", 400)
			return
		}

		//query
		rows, err := database.Query("SELECT * FROM " + table)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()

		//getcolumns
		columns, err := rows.Columns()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		//eader set (download)
		w.Header().Set("Content-Disposition", "attachment; filename="+table+".csv")
		w.Header().Set("Content-Type", "text/csv")

		writer := csv.NewWriter(w)
		defer writer.Flush()

		//write header
		writer.Write(columns)

		//prepare values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		//looprows
		for rows.Next() {

			err := rows.Scan(valuePtrs...)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			record := make([]string, len(columns))

			for i, val := range values {
				if val == nil {
					record[i] = ""
					continue
				}

				switch v := val.(type) {
				case []byte:
					record[i] = string(v)
				case string:
					record[i] = v
				default:
					record[i] = fmt.Sprintf("%v", v)
				}
			}

			writer.Write(record)
		}
	})

	//csvimport(route add)

	http.HandleFunc("/api/import/", func(w http.ResponseWriter, r *http.Request) {

		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

		table := strings.TrimPrefix(r.URL.Path, "/api/import/")
		table = strings.TrimSuffix(table, "/csv")

		allowed := map[string]bool{
			"users": true,
		}

		if !allowed[table] {
			http.Error(w, "invalid table", 400)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer file.Close()

		reader := csv.NewReader(file)

		headers, err := reader.Read()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		//DB columns
		cols, err := database.Query("DESCRIBE " + table)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer cols.Close()

		dbColumns := []string{}
		dbTypes := []string{}
		requiredCols := []string{}

		for cols.Next() {
			var field, colType, null, key, extra string
			var defaultVal *string

			cols.Scan(&field, &colType, &null, &key, &defaultVal, &extra)

			dbColumns = append(dbColumns, field)
			dbTypes = append(dbTypes, colType)

			if null == "NO" && key != "PRI" {
				requiredCols = append(requiredCols, field)
			}
		}
		for _, h := range headers {
			found := false
			for _, dbCol := range dbColumns {
				if h == dbCol {
					found = true
				}
			}
			if !found {
				http.Error(w, "unknown column: "+h, 400)
				return
			}
		}

		//column check
		for _, req := range requiredCols {
			found := false
			for _, h := range headers {
				if h == req {
					found = true
				}
			}
			if !found {
				http.Error(w, "missing required column: "+req, 400)
				return
			}
		}

		placeholders := make([]string, len(headers))
		for i := range headers {
			placeholders[i] = "?"
		}

		query := fmt.Sprintf(
			"INSERT INTO `%s` (%s) VALUES (%s)",
			table,
			strings.Join(headers, ","),
			strings.Join(placeholders, ","),
		)

		tx, err := database.Begin()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		total := 0
		inserted := 0
		failed := 0
		errorsList := []string{}

		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				tx.Rollback()
				http.Error(w, err.Error(), 500)
				return
			}

			total++
			values := make([]interface{}, len(record))

			for i, v := range record {

				var colType string

				// match header with db column type
				for j, col := range dbColumns {
					if headers[i] == col {
						colType = dbTypes[j]
						break
					}
				}

				converted, err := convertValue(v, colType)
				if err != nil {
					failed++
					errorsList = append(errorsList,
						fmt.Sprintf("row %d failed: invalid type for column %s", total, headers[i]))
					continue
				}

				values[i] = converted
			}

			_, err = tx.Exec(query, values...)
			if err != nil {
				failed++
				errorsList = append(errorsList,
					fmt.Sprintf("row %d failed: %v", total, err))
				continue
			}

			inserted++
		}

		//commit
		err = tx.Commit()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		//response
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total":    total,
			"inserted": inserted,
			"failed":   failed,
			"errors":   errorsList,
		})
	})

	fmt.Println(cfg.PORT)
	fmt.Println(cfg.DBUSER)
	fmt.Println(cfg.DBPASSWORD)
	fmt.Println(cfg.DBHOST)
	fmt.Println(cfg.DBPORT)
	fmt.Println(cfg.DBNAME)
	fmt.Println("app started")

	log.Println("server running on :5000")
	log.Fatal(http.ListenAndServe(":5000", nil))

}

func importExcelHandler(w http.ResponseWriter, r *http.Request) {

	// limit file size
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	// table extract
	table := strings.TrimPrefix(r.URL.Path, "/api/import-excel/")

	allowed := map[string]bool{
		"users": true,
	}

	if !allowed[table] {
		http.Error(w, "invalid table", 400)
		return
	}

	// read file
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer file.Close()

	// open excel
	f, err := excelize.OpenReader(file)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if len(rows) < 1 {
		http.Error(w, "empty file", 400)
		return
	}

	headers := rows[0]

	// DB schema
	cols, err := db.DB.Query("DESCRIBE " + table)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer cols.Close()

	dbColumns := []string{}
	dbTypes := []string{}
	requiredCols := []string{}

	for cols.Next() {
		var field, colType, null, key, extra string
		var defaultVal *string

		cols.Scan(&field, &colType, &null, &key, &defaultVal, &extra)

		dbColumns = append(dbColumns, field)
		dbTypes = append(dbTypes, colType)

		if null == "NO" && key != "PRI" {
			requiredCols = append(requiredCols, field)
		}
	}

	// unknown column check
	for _, h := range headers {
		found := false
		for _, dbCol := range dbColumns {
			if h == dbCol {
				found = true
			}
		}
		if !found {
			http.Error(w, "unknown column: "+h, 400)
			return
		}
	}

	// required column check
	for _, req := range requiredCols {
		found := false
		for _, h := range headers {
			if h == req {
				found = true
			}
		}
		if !found {
			http.Error(w, "missing required column: "+req, 400)
			return
		}
	}

	// placeholders
	placeholders := make([]string, len(headers))
	for i := range headers {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(
		"INSERT INTO `%s` (%s) VALUES (%s)",
		table,
		strings.Join(headers, ","),
		strings.Join(placeholders, ","),
	)

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	total := 0
	inserted := 0
	failed := 0
	errorsList := []string{}

	// loop rows (skip header)
	for i, row := range rows[1:] {

		total++

		values := make([]interface{}, len(headers))

		for j, v := range row {

			var colType string

			// match excel header with db column type
			for k, col := range dbColumns {
				if headers[j] == col {
					colType = dbTypes[k]
					break
				}
			}

			converted, err := convertValue(v, colType)
			if err != nil {
				failed++
				errorsList = append(errorsList,
					fmt.Sprintf("row %d failed: invalid type for column %s", i+1, headers[j]))
				continue
			}

			values[j] = converted
		}

		_, err := tx.Exec(query, values...)
		if err != nil {
			failed++
			errorsList = append(errorsList,
				fmt.Sprintf("row %d failed: %v", i+1, err))
			continue
		}

		inserted++
	}

	err = tx.Commit()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"total":    total,
		"inserted": inserted,
		"failed":   failed,
		"errors":   errorsList,
	})
}

//type conversion add

func convertValue(value string, colType string) (interface{}, error) {

	if value == "" {
		return nil, nil
	}

	if strings.Contains(colType, "int") {
		return strconv.Atoi(value)
	}

	if strings.Contains(colType, "float") {
		return strconv.ParseFloat(value, 64)
	}

	return value, nil
}
