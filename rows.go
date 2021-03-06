package dbs

import (
	"database/sql"
	"errors"
	"reflect"
	"time"
)

const (
	k_SQL_TAG    = "sql"
	k_SQL_NO_TAG = "-"
)

func Scan(rows *sql.Rows, dest interface{}) (err error) {
	if rows == nil {
		return errors.New("rows: rows is closed")
	}

	var destType = reflect.TypeOf(dest)
	var destValue = reflect.ValueOf(dest)
	var destValueKind = destValue.Kind()

	if destValueKind == reflect.Struct {
		return errors.New("rows: dest argument is struct")
	}

	if destValue.IsNil() {
		return errors.New("rows: dest argument is nil")
	}

	// 获取查询的列
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	//var hasData = false
	var isInit = false
	var isSlice = false
	var sliceValue reflect.Value

	for rows.Next() {
		//hasData = true

		if !isInit {
			for {
				if destValueKind == reflect.Ptr && destValue.IsNil() {
					destValue.Set(reflect.New(destType.Elem()))
				}

				if destValueKind == reflect.Ptr {
					destValue = destValue.Elem()
					destType = destType.Elem()
					destValueKind = destValue.Kind()
					continue
				}
				break
			}
			isInit = true

			if destValueKind == reflect.Slice {
				isSlice = true

				if destValue.IsValid() {
					destValue.Set(reflect.MakeSlice(destType, 0, 0))
				}

				sliceValue = destValue
			}
		}

		if isSlice {
			var obj = reflect.New(sliceValue.Type().Elem())

			err = _scan(rows, columns, obj.Interface())
			if err != nil {
				return err
			}

			sliceValue = reflect.Append(sliceValue, obj.Elem())
		} else {
			return _scan(rows, columns, dest)
		}
	}

	if isSlice {
		destValue.Set(sliceValue)
	}

	if e := rows.Err(); e != nil {
		return e
	}

	//if !hasData {
	//	return errors.New("rows: no rows in result set")
	//}

	return err
}

func _scan(rows *sql.Rows, columns []string, dest interface{}) (err error) {
	var destType = reflect.TypeOf(dest)
	var destValue = reflect.ValueOf(dest)
	var destValueKind = destValue.Kind()

	if destValueKind == reflect.Struct {
		return errors.New("rows: dest argument is struct")
	}

	if destValue.IsNil() {
		return errors.New("rows: dest argument is nil")
	}

	for {
		if destValueKind == reflect.Ptr && destValue.IsNil() {
			destValue.Set(reflect.New(destType.Elem()))
		}

		if destValueKind == reflect.Ptr {
			destValue = destValue.Elem()
			destType = destType.Elem()
			destValueKind = destValue.Kind()
			continue
		}
		break
	}

	var fields = make(map[string]*field)
	getFields(fields, destType, destValue)

	var valueList = make([]interface{}, 0, len(columns))
	var selectedFields = make([]*field, 0, len(columns))

	for _, column := range columns {
		if f, ok := fields[column]; ok {
			if f.Value.Kind() == reflect.Ptr {
				valueList = append(valueList, f.Value.Addr().Interface())
			} else {
				var value = reflect.New(reflect.PtrTo(f.Struct.Type))
				value.Elem().Set(f.Value.Addr())

				valueList = append(valueList, value.Interface())
			}
			selectedFields = append(selectedFields, f)
		}
	}

	if err = rows.Scan(valueList...); err != nil {
		return err
	}

	for index, f := range selectedFields {
		var v = reflect.ValueOf(valueList[index]).Elem().Elem()
		if v.IsValid() {
			if f.Value.Kind() == reflect.Ptr {
				f.Value.Set(v.Addr())
			} else {
				f.Value.Set(v)
			}
		}
	}

	return err
}

func getFields(fields map[string]*field, objType reflect.Type, objValue reflect.Value) {
	var numField = objType.NumField()

	for i := 0; i < numField; i++ {
		var fieldStruct = objType.Field(i)
		var fieldValue = objValue.Field(i)

		var tag = fieldStruct.Tag.Get(k_SQL_TAG)

		if tag == k_SQL_NO_TAG {
			continue
		}

		if tag == "" {
			if fieldValue.Kind() == reflect.Ptr {
				if fieldValue.Type() == reflect.TypeOf(&time.Time{}) {
					continue
				}
				if fieldValue.IsNil() {
					fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
				}
				fieldValue = fieldValue.Elem()
			}

			if fieldValue.Kind() == reflect.Struct {
				if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
					continue
				}
				getFields(fields, fieldValue.Type(), fieldValue)
				continue
			}
			tag = fieldStruct.Name
		}

		var f = &field{}
		f.Value = fieldValue
		f.Struct = fieldStruct
		f.Name = tag

		fields[tag] = f
	}
}

type field struct {
	Name   string
	Struct reflect.StructField
	Value  reflect.Value
}
