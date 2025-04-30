package llm

import (
	"database/sql"
	"fmt"
	"sort"

	_ "github.com/mattn/go-sqlite3"

	"github.com/rs/zerolog/log"
)

const SysPrompt = `You are an expert SQL query generator. Your task is to convert natural language questions into accurate, efficient SQL queries based on the provided PostgreSQL database schema. You will be provided with a database schema for every question. Follow these guidelines:
Only use the tables, columns, and relationships explicitly defined in the schema.
Use proper SQL syntax for PostgreSQL (e.g., ILIKE for case-insensitive matches, LIMIT for result size).`

func getSchemas(dbId string) string {
	db, err := sql.Open("sqlite3", fmt.Sprintf("/data/%s.sqlite", dbId))
	if err != nil {
		log.Panic().Err(err).Msg("Failed to open database")
	}
	defer db.Close()

	// Query
	rows, err := db.Query(`SELECT sql FROM sqlite_master where type='table'`)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to open database")
	}
	defer rows.Close()
	schemas := ""
	for rows.Next() {
		var tableSchema string
		rows.Scan(&tableSchema)
		schemas += fmt.Sprintf("%s\n\n", tableSchema)
	}
	log.Debug().Str("schemas", schemas).Msg("schemas")
	return schemas
}

func compareRows(goldRows, llmRows *sql.Rows) (bool, error) {
	goldMaps, err := rowsToMaps(goldRows)
	if err != nil {
		return false, err
	}

	llmMaps, err := rowsToMaps(llmRows)
	if err != nil {
		return false, err
	}

	return compareResultSets(goldMaps, llmMaps), nil
}

func rowsToMaps(rows *sql.Rows) ([]map[string]any, error) {
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]any

	for rows.Next() {
		values := make([]any, len(cols))
		pointers := make([]any, len(cols))
		for i := range values {
			pointers[i] = &values[i]
		}

		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]any)
		for i, col := range cols {
			rowMap[col] = values[i]
		}

		result = append(result, rowMap)
	}

	return result, nil
}

func compareResultSets(set1, set2 []map[string]any) bool {
	if len(set1) != len(set2) {
		return false
	}

	// Sort by JSON string for consistent comparison
	sortFunc := func(data []map[string]any) {
		sort.Slice(data, func(i, j int) bool {
			return fmt.Sprintf("%v", data[i]) < fmt.Sprintf("%v", data[j])
		})
	}
	sortFunc(set1)
	sortFunc(set2)

	for i := range set1 {
		if fmt.Sprintf("%v", set1[i]) != fmt.Sprintf("%v", set2[i]) {
			return false
		}
	}

	return true
}
