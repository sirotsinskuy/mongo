package querybuilder

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	reNull = regexp.MustCompile(`null`)
	reWord = regexp.MustCompile(`\p{L}+$|[0-9]+`)
)

func detectDateComparisonOperator(field string, values []string) bson.M {
	if len(values) == 0 {
		return nil
	}

	// if values is greater than 0, use an $in/$nin clause
	if len(values) > 1 {
		a := bson.A{}

		parsedValues, operator := mustDetectNotInOperator(values)

		// add each string value to the bson.A
		rangeFilterUsed := false
		for _, v := range parsedValues {
			if strings.HasPrefix(v, "><") {
				rangeFilterUsed = true
				v = strings.TrimPrefix(v, "><")
			}

			dv, err := time.Parse(time.RFC3339, v)
			if err != nil {
				panic(err)
			}
			fmt.Println(dv)
			a = append(a, dv)
		}

		// return a filter with the array of values...
		if rangeFilterUsed && len(a) == 2 {
			return bson.M{
				field: bson.D{
					primitive.E{
						Key:   "$gte",
						Value: a[0],
					},
					primitive.E{
						Key:   "$lte",
						Value: a[1],
					},
				},
			}
		} else {
			fmt.Println("not range")
		}

		// create a filter with the array of values...
		filter := bson.M{
			field: bson.D{primitive.E{
				Key:   operator,
				Value: a,
			}},
		}

		// return
		return filter
	}

	value := values[0]
	var oper string

	// check if string value is long enough for a 2 char prefix
	if len(value) >= 3 {
		var uv string

		// lte
		if value[0:2] == "<=" {
			oper = "$lte"
			uv = value[2:]
		}

		// gte
		if value[0:2] == ">=" {
			oper = "$gte"
			uv = value[2:]
		}

		// ne
		if value[0:2] == "!=" {
			oper = "$ne"
			uv = value[2:]
		}

		// update value to remove the prefix
		if uv != "" {
			value = uv
		}
	}

	// check if string value is long enough for a single char prefix
	if len(value) >= 2 {
		var uv string

		// lt
		if value[0:1] == "<" {
			oper = "$lt"
			uv = value[1:]
		}

		// gt
		if value[0:1] == ">" {
			oper = "$gt"
			uv = value[1:]
		}

		// ne
		if value[0:1] == "-" {
			oper = "$ne"
			uv = value[1:]
		}

		// update value to remove the prefix
		if uv != "" {
			value = uv
		}
	}

	// detect usage of keyword "null"
	if reNull.MatchString(value) {
		// check if there is an lt, lte, gt or gte key
		if oper != "" {
			return bson.M{field: bson.D{primitive.E{
				Key:   oper,
				Value: nil,
			}}}
		}

		// return the filter
		return bson.M{field: nil}
	}

	// parse the date value
	dv, _ := time.Parse(time.RFC3339, value)

	// check if there is an lt, lte, gt or gte key
	if oper != "" {
		return bson.M{field: bson.D{primitive.E{
			Key:   oper,
			Value: dv,
		}}}
	}

	// return the filter
	return bson.M{field: dv}
}

// mustDetectNotInOperator detects $in for all positive VS $nin for all negative values
func mustDetectNotInOperator(values []string) (updatedValues []string, operator string) {
	operator = "$in"

	notInCnt := 0
	for _, value := range values {
		if strings.HasPrefix(value, "-") {
			operator = "$nin"
			notInCnt++
		}
		updatedValues = append(updatedValues, strings.TrimPrefix(value, "-"))
	}
	if notInCnt > 0 && notInCnt != len(values) {
		panic("all elements must me either positive or negative")
	}

	return updatedValues, operator
}

func detectNumericComparisonOperator(field string, values []string, numericType string) bson.M {
	if len(values) == 0 {
		return nil
	}

	var bitSize int
	switch numericType {
	case "decimal":
		bitSize = 32
	case "double":
		bitSize = 64
	case "int":
		bitSize = 32
	case "long":
		bitSize = 64
	default:
		return nil
	}

	// handle when values is an array
	if len(values) > 1 {
		rangeFilterUsed := false
		a := bson.A{}

		for _, value := range values {
			if strings.HasPrefix(value, "><") {
				rangeFilterUsed = true
				value = strings.TrimPrefix(value, "><")
			}
			var pv interface{}
			if numericType == "decimal" || numericType == "double" {
				v, _ := strconv.ParseFloat(value, bitSize)
				pv = v

				// retype 32 bit
				if bitSize == 32 {
					pv = float32(v)
				}
			} else {
				v, _ := strconv.ParseInt(value, 0, bitSize)
				pv = v

				// retype 32 bit
				if bitSize == 32 {
					pv = int32(v)
				}
			}

			a = append(a, pv)
		}

		// return a filter with the array of values...
		if rangeFilterUsed && len(a) == 2 {
			return bson.M{
				field: bson.D{
					primitive.E{
						Key:   "$gte",
						Value: a[0],
					},
					primitive.E{
						Key:   "$lte",
						Value: a[1],
					},
				},
			}
		}
		return bson.M{
			field: bson.D{primitive.E{
				Key:   "$in",
				Value: a,
			}},
		}
	}

	var oper string
	value := values[0]

	// check if string value is long enough for a 2 char prefix
	if len(value) >= 3 {
		var uv string

		// lte
		if value[0:2] == "<=" {
			oper = "$lte"
			uv = value[2:]
		}

		// gte
		if value[0:2] == ">=" {
			oper = "$gte"
			uv = value[2:]
		}

		// ne
		if value[0:2] == "!=" {
			oper = "$ne"
			uv = value[2:]
		}

		// update value to remove the prefix
		if uv != "" {
			value = uv
		}
	}

	// check if string value is long enough for a single char prefix
	if len(value) >= 2 {
		var uv string

		// lt
		if value[0:1] == "<" {
			oper = "$lt"
			uv = value[1:]
		}

		// gt
		if value[0:1] == ">" {
			oper = "$gt"
			uv = value[1:]
		}

		// update value to remove the prefix
		if uv != "" {
			value = uv
		}
	}

	if reNull.MatchString(value) {
		// detect $ne operator (note use of - shorthand here which is not
		// processed on numeric values that are not "null")
		if value[0:1] == "-" || value[0:2] == "!=" {
			oper = "$ne"
		}

		if oper != "" {
			// return with the specified operator
			return bson.M{field: bson.D{primitive.E{
				Key:   oper,
				Value: nil,
			}}}
		}

		return bson.M{field: nil}
	}

	// parse the numeric value appropriately
	var parsedValue interface{}
	if numericType == "decimal" || numericType == "double" {
		v, _ := strconv.ParseFloat(value, bitSize)
		parsedValue = v

		// retype 32 bit
		if bitSize == 32 {
			parsedValue = float32(v)
		}
	}

	if parsedValue == nil {
		v, _ := strconv.ParseInt(value, 0, bitSize)
		parsedValue = v

		// retype 32 bit
		if bitSize == 32 {
			parsedValue = int32(v)
		}
	}

	// check if there is an lt, lte, gt or gte key
	if oper != "" {
		// return with the specified operator
		return bson.M{field: bson.D{primitive.E{
			Key:   oper,
			Value: parsedValue,
		}}}
	}

	// no operator... just the value
	return bson.M{field: parsedValue}
}

func detectStringComparisonOperator(field string, values []string, bsonType string) bson.M {
	if len(values) == 0 {
		return nil
	}

	// if bsonType is object, query should use an exists operator
	if bsonType == "object" {
		filter := bson.M{}

		for _, fn := range values {
			// check for "-" prefix on field name
			exists := true
			if len(fn) >= 2 && fn[0:1] == "-" {
				exists = false
				fn = fn[1:]
			}

			// check for "!=" prefix on field name
			// NOTE: this is a bit of an odd syntax, but support was simple
			// to build in
			if exists && len(fn) >= 3 && fn[0:2] == "!=" {
				exists = false
				fn = fn[2:]
			}

			fn = fmt.Sprintf("%s.%s", field, fn)
			filter[fn] = bson.D{primitive.E{
				Key:   "$exists",
				Value: exists,
			}}
		}

		return filter
	}

	// if values is greater than 0, use an $in clause
	if len(values) > 1 {
		a := bson.A{}

		// add each string value to the bson.A
		for _, v := range values {
			a = append(a, v)
		}

		// when type is an array, don't use $in operator
		if bsonType == "array" {
			return bson.M{field: a}
		}

		// create a filter with the array of values using an $in operator for strings...
		return bson.M{field: bson.D{primitive.E{
			Key:   "$in",
			Value: a,
		}}}
	}

	// single value
	value := values[0]

	// ensure we have a word/value to filter with
	if !reWord.MatchString(value) {
		return nil
	}

	bw := false
	containsOperator := false
	em := false
	ew := false
	ne := false
	orOperator := false

	if value[0:2] == "||" {
		orOperator = true
		value = value[2:]
	}

	// check for prefix/suffix on the value string
	if len(value) > 1 {
		bw = value[len(value)-1:] == "*"
		ew = value[0:1] == "*"
		containsOperator = bw && ew
		ne = value[0:1] == "-"

		// adjust value when not equal...
		if ne || ew {
			value = value[1:]
		}

		if bw {
			value = value[0 : len(value)-1]
		}

		if containsOperator {
			bw = false
			ew = false
		}
	}

	// "OR" handling
	if orOperator {
		//TODO handle other regexp cases as well
		if containsOperator {
			return bson.M{"$or": bson.M{
				field: primitive.Regex{
					Pattern: value,
					Options: "i",
				},
			}}
		}
		return bson.M{"$or": bson.M{
			field: value,
		}}
	}

	// check for != or string in quotes
	if len(value) > 2 && !ne {
		ne = value[0:2] == "!="
		em = value[0:1] == "\"" &&
			value[len(value)-1:] == "\""

		if ne {
			value = value[2:]
		}

		if em {
			value = value[1 : len(value)-1]
		}
	}

	// handle null keyword
	if reNull.MatchString(value) {
		if ne {
			return bson.M{field: bson.D{primitive.E{
				Key:   "$ne",
				Value: nil,
			}}}
		}

		return bson.M{field: nil}
	}

	// not equal...
	if ne {
		return bson.M{field: bson.D{primitive.E{
			Key:   "$ne",
			Value: value,
		}}}
	}

	// contains...
	if containsOperator {
		return bson.M{field: primitive.Regex{
			Pattern: value,
			Options: "i",
		}}
	}

	// begins with...
	if bw {
		return bson.M{field: primitive.Regex{
			Pattern: fmt.Sprintf("^%s", value),
			Options: "i",
		}}
	}

	// ends with...
	if ew {
		return bson.M{field: primitive.Regex{
			Pattern: fmt.Sprintf("%s$", value),
			Options: "i",
		}}
	}

	// exact match...
	if em {
		return bson.M{field: primitive.Regex{
			Pattern: fmt.Sprintf("^%s$", value),
			Options: "",
		}}
	}

	// the string value as is...
	return bson.M{field: value}
}

func combine(a bson.M, b bson.M) bson.M {
	for k, v := range b {
		if k == "$or" {
			if a[k] == nil {
				a[k] = bson.A{}
			}
			a[k] = append(a[k].(bson.A), v)
			continue
		}
		a[k] = v
	}

	return a
}
