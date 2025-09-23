package orm

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/cuemby/gor/pkg/gor"
)

// gorTransaction implements the gor.Transaction interface
type gorTransaction struct {
	tx  *sql.Tx
	orm *gorORM
}

// Commit commits the transaction
func (t *gorTransaction) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction
func (t *gorTransaction) Rollback() error {
	return t.tx.Rollback()
}

// Create creates a new record within the transaction
func (t *gorTransaction) Create(model interface{}) error {
	// Call BeforeCreate hook if model implements it
	if hook, ok := model.(interface{ BeforeCreate() error }); ok {
		if err := hook.BeforeCreate(); err != nil {
			return err
		}
	}

	// Set timestamps
	setTimestamps(model, true)

	// Generate insert SQL
	tableName := getTableName(reflect.TypeOf(model))
	fields, values, placeholders := t.extractFieldsAndValues(model, true)

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		fields,
		placeholders)

	result, err := t.tx.Exec(sql, values...)
	if err != nil {
		return err
	}

	// Set ID if auto-increment
	if id, err := result.LastInsertId(); err == nil {
		setID(model, id)
	}

	// Call AfterCreate hook if model implements it
	if hook, ok := model.(interface{ AfterCreate() error }); ok {
		if err := hook.AfterCreate(); err != nil {
			return err
		}
	}

	return nil
}

// Update updates an existing record within the transaction
func (t *gorTransaction) Update(model interface{}) error {
	// Call BeforeUpdate hook
	if hook, ok := model.(interface{ BeforeUpdate() error }); ok {
		if err := hook.BeforeUpdate(); err != nil {
			return err
		}
	}

	// Set updated timestamp
	setTimestamps(model, false)

	// Generate update SQL
	tableName := getTableName(reflect.TypeOf(model))
	setParts, values := t.extractUpdateFields(model)
	id := getID(model)

	sql := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", tableName, setParts)
	values = append(values, id)

	_, err := t.tx.Exec(sql, values...)
	if err != nil {
		return err
	}

	// Call AfterUpdate hook
	if hook, ok := model.(interface{ AfterUpdate() error }); ok {
		if err := hook.AfterUpdate(); err != nil {
			return err
		}
	}

	return nil
}

// Delete deletes a record within the transaction
func (t *gorTransaction) Delete(model interface{}) error {
	// Call BeforeDelete hook
	if hook, ok := model.(interface{ BeforeDelete() error }); ok {
		if err := hook.BeforeDelete(); err != nil {
			return err
		}
	}

	id := getID(model)
	tableName := getTableName(reflect.TypeOf(model))

	sql := fmt.Sprintf("DELETE FROM %s WHERE id = ?", tableName)
	_, err := t.tx.Exec(sql, id)
	if err != nil {
		return err
	}

	// Call AfterDelete hook
	if hook, ok := model.(interface{ AfterDelete() error }); ok {
		if err := hook.AfterDelete(); err != nil {
			return err
		}
	}

	return nil
}

// Query creates a new query builder within the transaction
func (t *gorTransaction) Query(model interface{}) gor.QueryBuilder {
	// Create a transaction-aware query builder
	return &TransactionQueryBuilder{
		QueryBuilder: NewQueryBuilder(model, nil, t.orm.adapter).(*QueryBuilder),
		tx:           t.tx,
	}
}

// Exec executes raw SQL within the transaction
func (t *gorTransaction) Exec(sqlQuery string, args ...interface{}) (sql.Result, error) {
	return t.tx.Exec(sqlQuery, args...)
}

// QuerySQL executes a raw SQL query within the transaction
func (t *gorTransaction) QuerySQL(sqlQuery string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.Query(sqlQuery, args...)
}

// QueryRow executes a raw SQL query that returns a single row within the transaction
func (t *gorTransaction) QueryRow(sqlQuery string, args ...interface{}) *sql.Row {
	return t.tx.QueryRow(sqlQuery, args...)
}

// TransactionQueryBuilder wraps QueryBuilder to use transaction
type TransactionQueryBuilder struct {
	*QueryBuilder
	tx *sql.Tx
}

// Override methods to use transaction
func (tqb *TransactionQueryBuilder) First(dest interface{}) error {
	tqb.limitValue = &[]int{1}[0]

	sql, args, err := tqb.adapter.GenerateSQL(tqb.QueryBuilder)
	if err != nil {
		return err
	}

	row := tqb.tx.QueryRow(sql, args...)
	return tqb.scanRow(row, dest)
}

func (tqb *TransactionQueryBuilder) FindAll(dest interface{}) error {
	sql, args, err := tqb.adapter.GenerateSQL(tqb.QueryBuilder)
	if err != nil {
		return err
	}

	rows, err := tqb.tx.Query(sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	return tqb.scanRows(rows, dest)
}

func (tqb *TransactionQueryBuilder) Count() (int64, error) {
	countQB := &QueryBuilder{
		model:           tqb.model,
		modelType:       tqb.modelType,
		tableName:       tqb.tableName,
		adapter:         tqb.adapter,
		whereConditions: tqb.whereConditions,
		whereArgs:       tqb.whereArgs,
		joins:           tqb.joins,
		isCount:         true,
	}

	sql, args, err := tqb.adapter.GenerateSQL(countQB)
	if err != nil {
		return 0, err
	}

	var count int64
	err = tqb.tx.QueryRow(sql, args...).Scan(&count)
	return count, err
}

func (tqb *TransactionQueryBuilder) UpdateAll(updates map[string]interface{}) (int64, error) {
	setParts := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+len(tqb.whereArgs))

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = ?", field))
		args = append(args, value)
	}

	sql := fmt.Sprintf("UPDATE %s SET %s", tqb.tableName, joinStrings(setParts, ", "))

	if len(tqb.whereConditions) > 0 {
		sql += " WHERE " + joinStrings(tqb.whereConditions, " AND ")
		args = append(args, tqb.whereArgs...)
	}

	result, err := tqb.tx.Exec(sql, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (tqb *TransactionQueryBuilder) DeleteAll() (int64, error) {
	sql := fmt.Sprintf("DELETE FROM %s", tqb.tableName)

	if len(tqb.whereConditions) > 0 {
		sql += " WHERE " + joinStrings(tqb.whereConditions, " AND ")
	}

	result, err := tqb.tx.Exec(sql, tqb.whereArgs...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// Helper functions for transaction
func (t *gorTransaction) extractFieldsAndValues(model interface{}, includeID bool) (string, []interface{}, string) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var fields []string
	var values []interface{}
	var placeholders []string

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		value := v.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip ID for inserts unless explicitly included
		if !includeID && field.Name == "ID" {
			continue
		}

		fieldName := toSnakeCase(field.Name)
		fields = append(fields, fieldName)
		values = append(values, value.Interface())
		placeholders = append(placeholders, "?")
	}

	return joinStrings(fields, ", "), values, joinStrings(placeholders, ", ")
}

func (t *gorTransaction) extractUpdateFields(model interface{}) (string, []interface{}) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var setParts []string
	var values []interface{}

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		value := v.Field(i)

		// Skip unexported fields and ID
		if !field.IsExported() || field.Name == "ID" {
			continue
		}

		fieldName := toSnakeCase(field.Name)
		setParts = append(setParts, fmt.Sprintf("%s = ?", fieldName))
		values = append(values, value.Interface())
	}

	return joinStrings(setParts, ", "), values
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}