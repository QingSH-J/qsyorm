package qsysession

import (
	"errors"
	"fmt"
	"qsyorm/qsyclause"
	"reflect"
	"strings"
)

// Insert adds a new record to the database
func (s *Session) Insert(values ...interface{}) (int64, error) {
	if len(values) == 0 || s.Schema == nil {
		return 0, errors.New("no values or schema provided")
	}

	// 调用 BeforeInsert 钩子
	for _, value := range values {
		if err := s.CallBeforeInsert(value); err != nil {
			return 0, err
		}
	}

	table := s.Schema.GetTableName()
	fields := make([]string, 0)
	vars := make([]interface{}, 0)

	// 调试输出: 打印Schema信息
	s.Logger.Info("Insert into table '%s', Schema has %d fields", table, len(s.Schema.Fields))
	s.Logger.Info("DbFieldToGo mapping contents:")
	for dbField, goField := range s.Schema.DbFieldToGo {
		s.Logger.Info("  DB field '%s' -> Go field '%s'", dbField, goField)
	}

	// Get field names and values from the provided struct
	for _, value := range values {
		reflectValue := reflect.Indirect(reflect.ValueOf(value))
		if reflectValue.Kind() != reflect.Struct {
			return 0, errors.New("value must be a struct")
		}

		// 调试输出: 打印结构体的所有字段和值
		s.Logger.Info("Struct type: %s", reflectValue.Type().Name())
		for i := 0; i < reflectValue.NumField(); i++ {
			field := reflectValue.Type().Field(i)
			s.Logger.Info("  Go field: %s, Value: %v", field.Name, reflectValue.Field(i).Interface())
		}

		for i, field := range s.Schema.Fields {
			// 跳过自增主键字段，让数据库自动处理
			if field.IsAutoIncrement && field.IsPrimaryKey {
				s.Logger.Info("  Skipping autoincrement primary key field: %s", field.Name)
				continue
			}
			fields = append(fields, field.Name)

			// 使用DbFieldToGo映射找到对应的Go结构体字段名
			goFieldName, ok := s.Schema.DbFieldToGo[field.Name]
			if !ok {
				// 如果映射中没有，尝试直接使用字段名
				goFieldName = field.Name
				s.Logger.Info("  [WARNING] No mapping found for DB field '%s', using as-is", field.Name)
			}

			// 调试输出: 打印字段映射和反射结果
			s.Logger.Info("  Processing field %d: DB field '%s' -> Go field '%s'", i, field.Name, goFieldName)
			fieldValue := reflectValue.FieldByName(goFieldName)
			if !fieldValue.IsValid() {
				s.Logger.Error("  [ERROR] Cannot find Go field '%s' in struct %s", goFieldName, reflectValue.Type().Name())
				// 尝试不区分大小写查找
				for j := 0; j < reflectValue.NumField(); j++ {
					structField := reflectValue.Type().Field(j)
					if strings.EqualFold(structField.Name, goFieldName) {
						s.Logger.Info("  [RECOVERY] Found case-insensitive match: '%s'", structField.Name)
						fieldValue = reflectValue.Field(j)
						break
					}
				}
				if !fieldValue.IsValid() {
					return 0, fmt.Errorf("cannot find field '%s' in struct %s", goFieldName, reflectValue.Type().Name())
				}
			}

			vars = append(vars, fieldValue.Interface())
			s.Logger.Info("  Added field value: %v", fieldValue.Interface())
		}
	}

	// Build the SQL statement
	builder := qsyclause.New()

	// Use the BuildInsert helper to create the INSERT clause
	sql, _ := qsyclause.BuildInsert(table, fields)
	builder.Set(qsyclause.INSERT, sql)

	// Combine all clauses
	sqlStr, sqlVars := builder.Build(qsyclause.INSERT)

	// 调试输出: 打印最终SQL和参数
	s.Logger.Info("Final SQL: %s", sqlStr)
	s.Logger.Info("Final SQL vars: %v, fields values: %v", sqlVars, vars)

	// Add the query to the session and execute it
	s.Raw(sqlStr, append(sqlVars, vars...)...)
	result, err := s.Exec()
	if err != nil {
		s.Logger.Error("Insert execution failed: %v", err)
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	// 调用 AfterInsert 钩子
	for _, value := range values {
		if err := s.CallAfterInsert(value); err != nil {
			return id, err
		}
	}

	return id, nil
}

// Find retrieves records from the database
func (s *Session) Find(dest interface{}, where string, vars ...interface{}) error {
	if s.Schema == nil {
		return errors.New("schema is nil")
	}

	// Ensure dest is a pointer to slice
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return errors.New("dest must be a pointer to slice")
	}

	// Get field names from schema
	fields := make([]string, 0, len(s.Schema.Fields))
	for _, field := range s.Schema.Fields {
		fields = append(fields, field.Name)
	}

	// Build the SQL statement
	builder := qsyclause.New()

	// Build the SELECT statement
	selectSql, _ := qsyclause.BuildSelect(s.Schema.GetTableName(), fields, "")
	builder.Set(qsyclause.SELECT, selectSql)

	// Add WHERE clause if provided
	if where != "" {
		whereSql, whereVars := qsyclause.BuildWhere(where, vars...)
		builder.Set(qsyclause.WHERE, whereSql)

		// Now we need to add the WHERE variables by rebuilding
		sqlStr, sqlVars := builder.Build(qsyclause.SELECT, qsyclause.WHERE)

		// Execute the query with the additional where vars
		s.Raw(sqlStr, append(sqlVars, whereVars...)...)
		rows, err := s.QueryRows()
		if err != nil {
			return err
		}
		defer rows.Close()

		// Element type of the slice
		elemType := destValue.Elem().Type().Elem()

		// Scan results into the destination slice
		for rows.Next() {
			// Create a new element of the slice type
			newElem := reflect.New(elemType).Elem()

			// Create a slice to hold the field addresses for scanning
			values := make([]interface{}, len(fields))
			for i, field := range s.Schema.Fields {
				// 使用DbFieldToGo映射找到对应的Go结构体字段名
				goFieldName, ok := s.Schema.DbFieldToGo[field.Name]
				if !ok {
					// 如果映射中没有，尝试直接使用字段名
					goFieldName = field.Name
				}

				fieldValue := newElem.FieldByName(goFieldName)
				values[i] = fieldValue.Addr().Interface()
			}

			// Scan the row into the values
			if err := rows.Scan(values...); err != nil {
				return err
			}

			// 对每个记录调用 AfterQuery 钩子
			newObj := newElem.Addr().Interface()
			if err := s.CallAfterQuery(newObj); err != nil {
				return err
			}

			// Append the new element to the result slice
			destValue.Elem().Set(reflect.Append(destValue.Elem(), newElem))
		}

		return rows.Err()
	} else {
		// No WHERE clause, just execute the SELECT
		sqlStr, sqlVars := builder.Build(qsyclause.SELECT)

		// Execute the query
		s.Raw(sqlStr, sqlVars...)
		rows, err := s.QueryRows()
		if err != nil {
			return err
		}
		defer rows.Close()

		// Element type of the slice
		elemType := destValue.Elem().Type().Elem()

		// Scan results into the destination slice
		for rows.Next() {
			// Create a new element of the slice type
			newElem := reflect.New(elemType).Elem()

			// Create a slice to hold the field addresses for scanning
			values := make([]interface{}, len(fields))
			for i, field := range s.Schema.Fields {
				// 使用DbFieldToGo映射找到对应的Go结构体字段名
				goFieldName, ok := s.Schema.DbFieldToGo[field.Name]
				if !ok {
					// 如果映射中没有，尝试直接使用字段名
					goFieldName = field.Name
				}

				fieldValue := newElem.FieldByName(goFieldName)
				values[i] = fieldValue.Addr().Interface()
			}

			// Scan the row into the values
			if err := rows.Scan(values...); err != nil {
				return err
			}

			// 对每个记录调用 AfterQuery 钩子
			newObj := newElem.Addr().Interface()
			if err := s.CallAfterQuery(newObj); err != nil {
				return err
			}

			// Append the new element to the result slice
			destValue.Elem().Set(reflect.Append(destValue.Elem(), newElem))
		}

		return rows.Err()
	}
}

// Update modifies existing records in the database
func (s *Session) Update(value interface{}, where string, vars ...interface{}) (int64, error) {
	if s.Schema == nil {
		return 0, errors.New("schema is nil")
	}

	// 调用 BeforeUpdate 钩子
	if err := s.CallBeforeUpdate(value); err != nil {
		return 0, err
	}

	// 获取字段名和值 - 在调用钩子后获取，这样钩子中的修改会被包含
	reflectValue := reflect.Indirect(reflect.ValueOf(value))
	if reflectValue.Kind() != reflect.Struct {
		return 0, errors.New("value must be a struct")
	}

	fields := make([]string, 0)
	updateVars := make([]interface{}, 0)

	for _, field := range s.Schema.Fields {
		// 排除自增主键字段
		if field.IsPrimaryKey && field.IsAutoIncrement {
			continue
		}
		fields = append(fields, field.Name)

		// 使用DbFieldToGo映射找到对应的Go结构体字段名
		goFieldName, ok := s.Schema.DbFieldToGo[field.Name]
		if !ok {
			// 如果映射中没有，尝试直接使用字段名
			goFieldName = field.Name
		}

		updateVars = append(updateVars, reflectValue.FieldByName(goFieldName).Interface())
	}

	// Build the SQL statement
	builder := qsyclause.New()

	// Create UPDATE clause
	updateSql, _ := qsyclause.BuildUpdate(s.Schema.GetTableName(), fields)
	builder.Set(qsyclause.UPDATE, updateSql)

	// Add WHERE clause if provided
	var whereVars []interface{}
	if where != "" {
		whereSql, wVars := qsyclause.BuildWhere(where, vars...)
		whereVars = wVars
		builder.Set(qsyclause.WHERE, whereSql)
	}

	// Build the query
	sqlStr, sqlVars := builder.Build(qsyclause.UPDATE, qsyclause.WHERE)

	// Execute the query - first add update vars, then where vars
	allVars := append(updateVars, sqlVars...)
	if where != "" {
		allVars = append(allVars, whereVars...)
	}
	s.Raw(sqlStr, allVars...)
	result, err := s.Exec()
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	// 调用 AfterUpdate 钩子
	if err := s.CallAfterUpdate(value); err != nil {
		return affected, err
	}

	return affected, nil
}

// Delete removes records from the database
func (s *Session) Delete(where string, vars ...interface{}) (int64, error) {
	if s.Schema == nil {
		return 0, errors.New("schema is nil")
	}

	// 创建一个模型实例来调用钩子（如果有）
	if s.Schema.Model != nil {
		if err := s.CallBeforeDelete(s.Schema.Model); err != nil {
			return 0, err
		}
	}

	// Build the SQL statement
	builder := qsyclause.New()

	// Create DELETE clause
	deleteSql, _ := qsyclause.BuildDelete(s.Schema.GetTableName())
	builder.Set(qsyclause.DELETE, deleteSql)

	// Add WHERE clause if provided
	var whereVars []interface{}
	if where != "" {
		whereSql, wVars := qsyclause.BuildWhere(where, vars...)
		whereVars = wVars
		builder.Set(qsyclause.WHERE, whereSql)
	}

	// Build the query
	sqlStr, sqlVars := builder.Build(qsyclause.DELETE, qsyclause.WHERE)

	// Execute the query with where vars
	allVars := sqlVars
	if where != "" {
		allVars = append(allVars, whereVars...)
	}
	s.Raw(sqlStr, allVars...)
	result, err := s.Exec()
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	// 调用 AfterDelete 钩子
	if s.Schema.Model != nil {
		if err := s.CallAfterDelete(s.Schema.Model); err != nil {
			return affected, err
		}
	}

	return affected, nil
}

// Count returns the number of records that match the condition
func (s *Session) Count(where string, vars ...interface{}) (int64, error) {
	if s.Schema == nil {
		return 0, errors.New("schema is nil")
	}

	// Build the SQL statement
	builder := qsyclause.New()

	// Create COUNT query
	countSql := "SELECT COUNT(*) FROM " + s.Schema.GetTableName()
	builder.Set(qsyclause.COUNT, countSql)

	// Add WHERE clause if provided
	var whereVars []interface{}
	if where != "" {
		whereSql, wVars := qsyclause.BuildWhere(where, vars...)
		whereVars = wVars
		builder.Set(qsyclause.WHERE, whereSql)
	}

	// Build the query
	sqlStr, sqlVars := builder.Build(qsyclause.COUNT, qsyclause.WHERE)

	// Execute the query with where vars
	allVars := sqlVars
	if where != "" {
		allVars = append(allVars, whereVars...)
	}
	s.Raw(sqlStr, allVars...)

	var count int64
	row := s.QueryRow()
	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}
