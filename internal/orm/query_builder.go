package orm

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/cuemby/gor/pkg/gor"
)

// QueryBuilder implements the gor.QueryBuilder interface
type QueryBuilder struct {
	model     interface{}
	modelType reflect.Type
	tableName string
	db        *sql.DB
	adapter   gor.DatabaseAdapter

	// Query building state
	whereConditions []string
	whereArgs       []interface{}
	orderBy         []string
	limitValue      *int
	offsetValue     *int
	joins           []string
	includes        []string
	isCount         bool
	rawSQL          string
	rawArgs         []interface{}
}

// NewQueryBuilder creates a new query builder instance
func NewQueryBuilder(model interface{}, db *sql.DB, adapter gor.DatabaseAdapter) gor.QueryBuilder {
	var modelType reflect.Type
	var tableName string

	if model != nil {
		modelType = reflect.TypeOf(model)
		if modelType.Kind() == reflect.Ptr {
			modelType = modelType.Elem()
		}
		if modelType.Kind() == reflect.Slice {
			modelType = modelType.Elem()
			if modelType.Kind() == reflect.Ptr {
				modelType = modelType.Elem()
			}
		}
		tableName = getTableName(modelType)
	}

	return &QueryBuilder{
		model:     model,
		modelType: modelType,
		tableName: tableName,
		db:        db,
		adapter:   adapter,
	}
}

// Where adds a WHERE condition
func (qb *QueryBuilder) Where(condition string, args ...interface{}) gor.QueryBuilder {
	qb.whereConditions = append(qb.whereConditions, condition)
	qb.whereArgs = append(qb.whereArgs, args...)
	return qb
}

// WhereMap adds multiple WHERE conditions from a map
func (qb *QueryBuilder) WhereMap(conditions map[string]interface{}) gor.QueryBuilder {
	for field, value := range conditions {
		qb.Where(fmt.Sprintf("%s = ?", field), value)
	}
	return qb
}

// Not adds a NOT WHERE condition
func (qb *QueryBuilder) Not(condition string, args ...interface{}) gor.QueryBuilder {
	qb.whereConditions = append(qb.whereConditions, fmt.Sprintf("NOT (%s)", condition))
	qb.whereArgs = append(qb.whereArgs, args...)
	return qb
}

// Order adds an ORDER BY clause
func (qb *QueryBuilder) Order(field string) gor.QueryBuilder {
	qb.orderBy = append(qb.orderBy, fmt.Sprintf("%s ASC", field))
	return qb
}

// OrderDesc adds an ORDER BY DESC clause
func (qb *QueryBuilder) OrderDesc(field string) gor.QueryBuilder {
	qb.orderBy = append(qb.orderBy, fmt.Sprintf("%s DESC", field))
	return qb
}

// Limit sets the LIMIT clause
func (qb *QueryBuilder) Limit(limit int) gor.QueryBuilder {
	qb.limitValue = &limit
	return qb
}

// Offset sets the OFFSET clause
func (qb *QueryBuilder) Offset(offset int) gor.QueryBuilder {
	qb.offsetValue = &offset
	return qb
}

// Page sets pagination (combination of limit and offset)
func (qb *QueryBuilder) Page(page, size int) gor.QueryBuilder {
	offset := (page - 1) * size
	qb.limitValue = &size
	qb.offsetValue = &offset
	return qb
}

// Joins adds a JOIN clause
func (qb *QueryBuilder) Joins(table string) gor.QueryBuilder {
	qb.joins = append(qb.joins, fmt.Sprintf("INNER JOIN %s", table))
	return qb
}

// LeftJoin adds a LEFT JOIN clause
func (qb *QueryBuilder) LeftJoin(table string) gor.QueryBuilder {
	qb.joins = append(qb.joins, fmt.Sprintf("LEFT JOIN %s", table))
	return qb
}

// Includes marks associations for eager loading
func (qb *QueryBuilder) Includes(associations ...string) gor.QueryBuilder {
	qb.includes = append(qb.includes, associations...)
	return qb
}

// Preload is an alias for Includes
func (qb *QueryBuilder) Preload(associations ...string) gor.QueryBuilder {
	return qb.Includes(associations...)
}

// Count returns the count of matching records
func (qb *QueryBuilder) Count() (int64, error) {
	countQB := &QueryBuilder{
		model:           qb.model,
		modelType:       qb.modelType,
		tableName:       qb.tableName,
		db:              qb.db,
		adapter:         qb.adapter,
		whereConditions: qb.whereConditions,
		whereArgs:       qb.whereArgs,
		joins:           qb.joins,
		isCount:         true,
	}

	sql, args, err := qb.adapter.GenerateSQL(countQB)
	if err != nil {
		return 0, err
	}

	var count int64
	err = qb.db.QueryRow(sql, args...).Scan(&count)
	return count, err
}

// Sum calculates the sum of a field
func (qb *QueryBuilder) Sum(field string) (float64, error) {
	sql := qb.buildAggregateSQL("SUM", field)
	args := qb.whereArgs

	var sum float64
	err := qb.db.QueryRow(sql, args...).Scan(&sum)
	return sum, err
}

// Average calculates the average of a field
func (qb *QueryBuilder) Average(field string) (float64, error) {
	sql := qb.buildAggregateSQL("AVG", field)
	args := qb.whereArgs

	var avg float64
	err := qb.db.QueryRow(sql, args...).Scan(&avg)
	return avg, err
}

// Maximum finds the maximum value of a field
func (qb *QueryBuilder) Maximum(field string) (interface{}, error) {
	sql := qb.buildAggregateSQL("MAX", field)
	args := qb.whereArgs

	var max interface{}
	err := qb.db.QueryRow(sql, args...).Scan(&max)
	return max, err
}

// Minimum finds the minimum value of a field
func (qb *QueryBuilder) Minimum(field string) (interface{}, error) {
	sql := qb.buildAggregateSQL("MIN", field)
	args := qb.whereArgs

	var min interface{}
	err := qb.db.QueryRow(sql, args...).Scan(&min)
	return min, err
}

// First finds the first matching record
func (qb *QueryBuilder) First(dest interface{}) error {
	qb.limitValue = &[]int{1}[0]

	sql, args, err := qb.adapter.GenerateSQL(qb)
	if err != nil {
		return err
	}

	row := qb.db.QueryRow(sql, args...)
	return qb.scanRow(row, dest)
}

// Last finds the last matching record
func (qb *QueryBuilder) Last(dest interface{}) error {
	// Reverse the order and get first
	if len(qb.orderBy) == 0 {
		qb.OrderDesc("id") // Default to ID DESC
	} else {
		// Reverse existing order
		for i, order := range qb.orderBy {
			if strings.Contains(order, "ASC") {
				qb.orderBy[i] = strings.Replace(order, "ASC", "DESC", 1)
			} else if strings.Contains(order, "DESC") {
				qb.orderBy[i] = strings.Replace(order, "DESC", "ASC", 1)
			}
		}
	}

	return qb.First(dest)
}

// Find is an alias for First
func (qb *QueryBuilder) Find(dest interface{}) error {
	return qb.First(dest)
}

// FindAll finds all matching records
func (qb *QueryBuilder) FindAll(dest interface{}) error {
	sql, args, err := qb.adapter.GenerateSQL(qb)
	if err != nil {
		return err
	}

	rows, err := qb.db.Query(sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	return qb.scanRows(rows, dest)
}

// Exists checks if any matching records exist
func (qb *QueryBuilder) Exists() (bool, error) {
	count, err := qb.Count()
	return count > 0, err
}

// FindInBatches processes records in batches
func (qb *QueryBuilder) FindInBatches(dest interface{}, batchSize int, fn func(tx gor.Transaction, batch interface{}) error) error {
	offset := 0

	for {
		batchQB := &QueryBuilder{
			model:           qb.model,
			modelType:       qb.modelType,
			tableName:       qb.tableName,
			db:              qb.db,
			adapter:         qb.adapter,
			whereConditions: qb.whereConditions,
			whereArgs:       qb.whereArgs,
			orderBy:         qb.orderBy,
			joins:           qb.joins,
			includes:        qb.includes,
			limitValue:      &batchSize,
			offsetValue:     &offset,
		}

		// Create batch slice
		batchType := reflect.TypeOf(dest)
		batch := reflect.New(batchType).Interface()

		err := batchQB.FindAll(batch)
		if err != nil {
			return err
		}

		// Check if batch is empty
		batchValue := reflect.ValueOf(batch).Elem()
		if batchValue.Len() == 0 {
			break
		}

		// Process batch in transaction
		tx, err := qb.db.Begin()
		if err != nil {
			return err
		}

		gorTx := &gorTransaction{tx: tx}
		if err := fn(gorTx, batch); err != nil {
			_ = tx.Rollback()
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		// If batch size is less than requested, we're done
		if batchValue.Len() < batchSize {
			break
		}

		offset += batchSize
	}

	return nil
}

// UpdateAll updates all matching records
func (qb *QueryBuilder) UpdateAll(updates map[string]interface{}) (int64, error) {
	setParts := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+len(qb.whereArgs))

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = ?", field))
		args = append(args, value)
	}

	sql := fmt.Sprintf("UPDATE %s SET %s", qb.tableName, strings.Join(setParts, ", "))

	if len(qb.whereConditions) > 0 {
		sql += " WHERE " + strings.Join(qb.whereConditions, " AND ")
		args = append(args, qb.whereArgs...)
	}

	result, err := qb.db.Exec(sql, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// DeleteAll deletes all matching records
func (qb *QueryBuilder) DeleteAll() (int64, error) {
	sql := fmt.Sprintf("DELETE FROM %s", qb.tableName)

	if len(qb.whereConditions) > 0 {
		sql += " WHERE " + strings.Join(qb.whereConditions, " AND ")
	}

	result, err := qb.db.Exec(sql, qb.whereArgs...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// Raw sets raw SQL query
func (qb *QueryBuilder) Raw(sqlQuery string, args ...interface{}) gor.QueryBuilder {
	qb.rawSQL = sqlQuery
	qb.rawArgs = args
	return qb
}

// Helper methods
func (qb *QueryBuilder) buildAggregateSQL(function, field string) string {
	sql := fmt.Sprintf("SELECT %s(%s) FROM %s", function, field, qb.tableName)

	if len(qb.joins) > 0 {
		sql += " " + strings.Join(qb.joins, " ")
	}

	if len(qb.whereConditions) > 0 {
		sql += " WHERE " + strings.Join(qb.whereConditions, " AND ")
	}

	return sql
}

func (qb *QueryBuilder) scanRow(row *sql.Row, dest interface{}) error {
	// Get the fields of the destination struct
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer")
	}

	destValue = destValue.Elem()
	destType := destValue.Type()

	// Prepare scan destinations
	scanDests := make([]interface{}, destType.NumField())
	for i := 0; i < destType.NumField(); i++ {
		field := destValue.Field(i)
		if field.CanSet() {
			scanDests[i] = field.Addr().Interface()
		}
	}

	return row.Scan(scanDests...)
}

func (qb *QueryBuilder) scanRows(rows *sql.Rows, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer to slice")
	}

	destValue = destValue.Elem()
	if destValue.Kind() != reflect.Slice {
		return fmt.Errorf("destination must be a pointer to slice")
	}

	elemType := destValue.Type().Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	for rows.Next() {
		// Create new instance
		elem := reflect.New(elemType)

		// Prepare scan destinations
		scanDests := make([]interface{}, elemType.NumField())
		for i := 0; i < elemType.NumField(); i++ {
			field := elem.Elem().Field(i)
			if field.CanSet() {
				scanDests[i] = field.Addr().Interface()
			}
		}

		if err := rows.Scan(scanDests...); err != nil {
			return err
		}

		// Append to slice
		if destValue.Type().Elem().Kind() == reflect.Ptr {
			destValue.Set(reflect.Append(destValue, elem))
		} else {
			destValue.Set(reflect.Append(destValue, elem.Elem()))
		}
	}

	return rows.Err()
}
