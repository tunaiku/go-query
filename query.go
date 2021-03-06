package query

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"
)

//Interface to provide functions to generate query
type providerQuery interface {
	ViewAll(table string) (query string, err error)
	Insert(table string) (query string, values []interface{}, err error)
	Delete(table string) (query string, err error)
	Update(primary, table string) (query string, values []interface{}, err error)
	UpdateWhere(primary, table string, primaryValue interface{}) (query string, values []interface{}, err error)
	Where(operatorCondition, operationBetweenCondition string) (query string, values []interface{}, err error)
}

/*
	Defining the body function
*/
func (s structModel) UpdateWhere(primary, table string, primaryValue interface{}) (query string, values []interface{}, err error) {
	if s.err != nil {
		return "", nil, s.err
	}
	var arrQuery []string
	query = "UPDATE " + table + " SET"
	listValues := make([]interface{}, 0)
	var totalIteraton int
	iteration := 1
	for i, _ := range s.value {
		if s.column[i] == primary {
			continue
		}
		arrQuery = append(arrQuery, " "+s.column[i]+"= $"+strconv.Itoa(iteration))
		listValues = append(listValues, s.value[i])
		iteration++
		totalIteraton = iteration
	}
	query = query + strings.Join(arrQuery, ",")
	query += " WHERE " + primary + "= $" + strconv.Itoa(totalIteraton)
	listValues = append(listValues, primaryValue)
	return query, listValues, nil
}

//Function to generating query for query SELECT *
func (s structModel) ViewAll(table string) (query string, err error) {
	if s.err != nil {
		return "", s.err
	}
	var arrQuery []string
	query = "SELECT"
	for i, _ := range s.value {
		arrQuery = append(arrQuery, " "+s.column[i])
	}
	query = query + strings.Join(arrQuery, ",") + " FROM " + table
	return query, nil
}

//Function to generating query for query INSERT
func (s structModel) Insert(table string) (query string, values []interface{}, err error) {
	if s.err != nil {
		return "", nil, s.err
	}
	var arrQuery, valArr []string
	query = "INSERT INTO " + table + "("
	queryForValues := " VALUES("
	listValue := make([]interface{}, 0)

	for i, _ := range s.value {
		arrQuery = append(arrQuery, " "+s.column[i])
		valArr = append(valArr, " $"+strconv.Itoa(i+1))
		listValue = append(listValue, s.value[i])
	}
	query = query + strings.Join(arrQuery, ",") + ")"
	queryForValues = queryForValues + strings.Join(valArr, ",") + ")"
	return query + queryForValues, listValue, nil
}

//Function to generating query for query DELETE
func (s structModel) Delete(table string) (query string, err error) {
	if s.err != nil {
		return "", s.err
	}

	query = "DELETE FROM " + table
	return query, nil
}

//Function to generating query for query UPDATE
func (s structModel) Update(primary, table string) (query string, values []interface{}, err error) {
	if s.err != nil {
		return "", nil, s.err
	}
	var arrQuery []string
	query = "UPDATE " + table + " SET"
	listValues := make([]interface{}, 0)
	for i, _ := range s.value {
		if s.column[i] == primary {
			continue
		}
		arrQuery = append(arrQuery, " "+s.column[i]+"= $"+strconv.Itoa(i+1))
		listValues = append(listValues, s.value[i])
	}
	query = query + strings.Join(arrQuery, ",")
	return query, listValues, nil
}

//Function to generating WHERE condition to the query
func (s structModel) Where(operatorCondition, operationBetweenCondition string) (query string, values []interface{}, err error) {
	if s.err != nil {
		return "", nil, s.err
	}
	query = " WHERE"
	listValues := make([]interface{}, 0)
	for i, _ := range s.value {
		query += " " + s.column[i] + " " + operatorCondition + " " + "$" + strconv.Itoa(i+1) + " " + operationBetweenCondition
		listValues = append(listValues, s.value[i])
	}
	query = query[0:(len(query) - len(operationBetweenCondition))]
	return query, listValues, nil
}

/*
	Model that is used as column and value to create the query
	column : database column name
	value : expected value
*/
type structModel struct {
	column []string
	value  []interface{}
	err    error
}

type batchQuery interface {
	InsertQuery(table string) (query string, err error)
	ValueBatch() (query string, values []interface{}, err error)
}

func (batchStructModel) InsertQuery(table string) (query string, err error) {

	return "", nil
}
func (batchStructModel) ValueBatch() (query string, values []interface{}, err error) {

	return "", nil, nil
}

type batchStructModel struct {
	values []structModel
	err    error
}

func ValueConversion(model interface{}) batchQuery {
	if reflect.TypeOf(model).Kind() == reflect.Slice {
		/*
			Check if inside the slice is a struct
		*/
		value := reflect.TypeOf(model).Elem()
		if reflect.TypeOf(value.Field(0)).Kind() != reflect.Struct {
			return batchQuery(batchStructModel{err: errors.New("parameter must be a struct")})
		}
	}
	convertedModel := batchStructModel{}
	result := batchQuery(convertedModel)
	return result
}

func Conversion(model interface{}) providerQuery {
	/*
		Returning error as validation to force function only accepting struct
		struct assume as reflect.Struct
	*/
	if reflect.TypeOf(model).Kind() != reflect.Struct {
		return providerQuery(structModel{err: errors.New("parameter must be a struct")})
	}

	var keys []string
	var vals []interface{}

	typeReflect := reflect.TypeOf(model)
	valReflect := reflect.ValueOf(model)
	/*
		Loop through the model to convert it to other model ('structModel')
		to be treated as column and value
	*/
	for i := 0; i < typeReflect.NumField(); i++ {
		typField := typeReflect.Field(i)
		valueField := valReflect.Field(i)
		/*
			Skip iteration if data is empty
			now empty is considered as empty string ("") or 0 if the data type is integer
		*/
		if typField.Type.Kind() == reflect.Struct {
			continue
		}
		if _, ok := typField.Tag.Lookup("skip"); ok {
			continue
		}
		keyValue, ok := typField.Tag.Lookup("db")
		if !ok {
			/*
				If tag not found
				the converion will search for default tag
				the default tag wll be used, to manipulate string in the struct's attribute
				with ToUpper or ToLower
				thus the Tag should be "lower" or "upper"
				other than that struct's attribute name will be used to indentify database column
			*/
			tagDefault, ok := typField.Tag.Lookup("default")
			if !ok {
				keyValue = typField.Name
			}
			switch tagDefault {
			case "lower":
				keyValue = strings.ToLower(typField.Name)
			case "upper":
				keyValue = strings.ToUpper(typField.Name)
			default:
				keyValue = typField.Name
			}
		}

		dateVal, ok := typField.Tag.Lookup("date")
		if ok {
			timeNow := time.Now()
			//insertTime,_ := time.Parse("2006-01-02 15:04:05", timeNow.String())
			if dateVal == "now" {
				keys = append(keys, keyValue)
				vals = append(vals, timeNow)
			} else if dateVal == "CURRENT_TIMESTAMP" {
				keys = append(keys, keyValue)
				vals = append(vals, "now")
			}
			continue
		}
		valueCase, ok := typField.Tag.Lookup("case")
		if ok {
			if valueCase == "upper" {
				keys = append(keys, keyValue)
				vals = append(vals, strings.ToUpper(valueField.String()))
			} else if valueCase == "lower" {
				keys = append(keys, keyValue)
				vals = append(vals, strings.ToLower(valueField.String()))
			}
			continue
		}
		keys = append(keys, keyValue)
		vals = append(vals, valueField.Interface().(interface{}))

	}
	convertedModel := structModel{column: keys, value: vals, err: nil}
	result := providerQuery(convertedModel)
	return result
}

type joinQuery interface {
	SelectAll(table []string, on []string) (query string, err error)
}

type joinModel struct {
	table []tableModel
	err   error
}

type tableModel struct {
	column []string
}

func (j joinModel) SelectAll(tables []string, on []string) (query string, err error) {
	if j.err != nil {
		return "", err
	}

	selectQuery := "SELECT"
	fromQuery := "FROM"
	var columns []string

	//iteration for columns
	for tableIteration := range j.table {
		currentTable := j.table[tableIteration]

		for columnIteration := range currentTable.column {
			columns = append(columns, currentTable.column[columnIteration])
		}
	}
	selectQuery = selectQuery + " " + strings.Join(columns, ",")
	fromQuery = " " + fromQuery + " " + strings.Join(tables, ",")
	whereClause := " WHERE " + strings.Join(on, " AND ")
	return selectQuery + fromQuery + whereClause, nil
}

func JoinClause(model ...interface{}) joinQuery {
	if len(model) < 1 {
		return joinQuery(joinModel{err: errors.New("model is not found")})
	}
	var result []tableModel
	for iteration := range model {
		typeReflect := reflect.TypeOf(model[iteration])
		var tempColumn []string
		for i := 0; i < typeReflect.NumField(); i++ {
			typeField := typeReflect.Field(i)
			columnName, ok := typeField.Tag.Lookup("db")
			if !ok {
				return joinQuery(joinModel{err: errors.New("tag db in model not found")})
			}
			tempColumn = append(tempColumn, columnName)
		}
		result = append(result, tableModel{column: tempColumn})
	}
	return joinQuery(joinModel{table: result, err: nil})
}

func handleSchemaAndTable(value string) string {
	dotIndex := strings.Index(value, ".")
	if dotIndex < 0 {
		return value
	}
	return value[dotIndex+1 : len(value)]
}
