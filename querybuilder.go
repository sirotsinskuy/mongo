package querybuilder

import (
	"fmt"
	"strconv"
	"strings"

	queryoptions "go.jtlabs.io/query"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// QueryBuilder is a type that makes working with Mongo driver Find methods easier
// when used in combination with a QueryOptions struct that specifies filters,
// pagination details, sorting instructions and field projection details.
type QueryBuilder struct {
	collection       string
	fieldTypes       map[string]string
	strictValidation bool
}

// NewQueryBuilder returns a new instance of a QueryBuilder object for constructing
// filters and options suitable for use with Mongo driver Find methods
func NewQueryBuilder(collection string, schema bson.M, strictValidation ...bool) *QueryBuilder {
	qb := QueryBuilder{
		collection:       collection,
		fieldTypes:       map[string]string{},
		strictValidation: false,
	}

	// parse the schema
	if schema != nil {
		qb.discoverFields(schema)
	}

	// override strict validation if provided
	if len(strictValidation) > 0 {
		qb.strictValidation = strictValidation[0]
	}

	return &qb
}

// Filter builds a suitable bson document to send to any of the find methods
// exposed by the Mongo driver. This method can validate the provided query
// options against the schema that was used to build the QueryBuilder instance
// when the QueryBuilder has strict validation enabled.
//
// The supported bson types for filter/search are:
// * array (strings only and not with $in operator unless sub items are strings)
// * bool
// * date
// * decimal
// * double
// * int
// * long
// * object (field detection)
// * string
// * timestamp
//
// The non-supported bson types for filter/search at this time
// * object (actual object comparison... only fields within the object are supported)
// * array (non string data)
// * binData
// * objectId
// * null
// * regex
// * dbPointer
// * javascript
// * symbol
// * javascriptWithScope
// * minKey
// * maxKey
func (qb QueryBuilder) Filter(qo queryoptions.Options) (bson.M, error) {
	filter := bson.M{}

	if len(qo.Filter) > 0 {
		for field, values := range qo.Filter {
			// handle array fields
			fiendNameWithNoIdx := strings.Split(field, "[]")[0]

			//elemMatchField := false
			if strings.Contains(field, ".[*]") {
				fiendNameWithNoIdx = strings.Split(field, ".[*]")[0]
				//elemMatchField = true
			}

			var bsonType string

			// lookup the field
			if bt, ok := qb.fieldTypes[fiendNameWithNoIdx]; ok {
				bsonType = bt
			}

			// check for strict field validation
			if bsonType == "" && qb.strictValidation {
				return nil, fmt.Errorf("field %s does not exist in collection %s", fiendNameWithNoIdx, qb.collection)
			}

			field = strings.ReplaceAll(field, "[]", ".")

			switch bsonType {
			case "array":
				f := detectStringComparisonOperator(field, values, bsonType)
				filter = combine(filter, f)
			case "bool":
				for _, value := range values {
					usedNe := false
					if strings.HasPrefix(value, "-") {
						usedNe = true
						value = strings.TrimPrefix(value, "-")
					}

					bv, _ := strconv.ParseBool(value)

					var f primitive.M
					if usedNe {
						f = primitive.M{field: primitive.M{
							"$ne": bv,
						}}
					} else {
						f = primitive.M{field: bv}
					}
					filter = combine(filter, f)
				}
			case "date":
				f := detectDateComparisonOperator(field, values)
				filter = combine(filter, f)
			case "decimal":
				f := detectNumericComparisonOperator(field, values, bsonType)
				filter = combine(filter, f)
			case "double":
				f := detectNumericComparisonOperator(field, values, bsonType)
				filter = combine(filter, f)
			case "int":
				f := detectNumericComparisonOperator(field, values, bsonType)
				filter = combine(filter, f)
			case "long":
				f := detectNumericComparisonOperator(field, values, bsonType)
				filter = combine(filter, f)
			case "object":
				f := detectStringComparisonOperator(field, values, bsonType)
				filter = combine(filter, f)
			case "string":
				f := detectStringComparisonOperator(field, values, bsonType)
				filter = combine(filter, f)
			case "timestamp":
				// handle just like dates
				f := detectDateComparisonOperator(field, values)
				filter = combine(filter, f)
			case "geo":
				f := detectGeoComparisonOperator(field, values)
				filter = combine(filter, f)
			}
		}
	}

	return filter, nil
}

func detectGeoComparisonOperator(field string, values []string) bson.M {
	switch len(values) {
	case 5:
		return processBoxOperator(field, values)
	case 3:
		lat, err := strconv.ParseFloat(values[0], 64)
		if err != nil {
			panic("incorrect value: part 1 is not float")
		}
		lon, err := strconv.ParseFloat(values[1], 64)
		if err != nil {
			panic("incorrect value: part 2 is not float")
		}
		radius, err := strconv.ParseFloat(values[2], 64)
		if err != nil {
			panic("incorrect value: part 3 is not float")
		}

		return bson.M{field: bson.M{
			"$nearSphere": bson.M{
				"$geometry": bson.M{
					"type":        "Point",
					"coordinates": []float64{lat, lon},
				},
				"$maxDistance": radius,
			}}}
	default:
		panic("incorrect value: not 3 or 4 val")
	}
}

func processBoxOperator(field string, values []string) bson.M {
	lat1, err := strconv.ParseFloat(values[0], 64)
	if err != nil {
		panic("incorrect value: part 1 is not float")
	}
	lon1, err := strconv.ParseFloat(values[1], 64)
	if err != nil {
		panic("incorrect value: part 2 is not float")
	}
	lat2, err := strconv.ParseFloat(values[2], 64)
	if err != nil {
		panic("incorrect value: part 3 is not float")
	}
	lon2, err := strconv.ParseFloat(values[3], 64)
	if err != nil {
		panic("incorrect value: part 4 is not float")
	}

	return bson.M{field: bson.M{
		"$geoWithin": bson.M{
			"$box": bson.A{
				[]float64{lat1, lon1},
				[]float64{lat2, lon2},
			},
		}}}
}

// FindOptions creates a mongo.FindOptions struct with pagination details, sorting,
// and field projection instructions set as specified in the query options input
func (qb QueryBuilder) FindOptions(qo queryoptions.Options) (*options.FindOptions, error) {
	opts := options.Find()

	// determine pagination for the options
	qb.setPaginationOptions(qo.Page, opts)

	// determine projection for the options
	if err := qb.setProjectionOptions(qo.Fields, opts); err != nil {
		return nil, err
	}

	// determine sorting for the options
	if err := qb.setSortOptions(qo.Sort, opts); err != nil {
		return nil, err
	}

	return opts, nil
}

func (qb QueryBuilder) discoverFields(schema bson.M) {
	// ensure fieldTypes is set
	if qb.fieldTypes == nil {
		qb.fieldTypes = map[string]string{}
	}

	// check to see if top level is $jsonSchema
	if js, ok := schema["$jsonSchema"]; ok {
		schema = js.(bson.M)
	}

	// bsonType, required, properties at top level
	// looking for properties field, specifically
	if properties, ok := schema["properties"]; ok {
		properties := properties.(bson.M)
		qb.iterateProperties("", properties)
	}
}

func (qb QueryBuilder) iterateProperties(parentPrefix string, properties bson.M) {
	// iterate each field within properties
	for field, value := range properties {
		switch value := value.(type) {
		case bson.M:
			// retrieve the type of the field
			if bsonType, ok := value["bsonType"]; ok {
				bsonType := bsonType.(string)
				// capture type in the fieldTypes map
				if bsonType != "" {
					qb.fieldTypes[fmt.Sprintf("%s%s", parentPrefix, field)] = bsonType
				}

				if bsonType == "array" {
					// look at "items" to get the bsonType
					if items, ok := value["items"]; ok {
						value = items.(bson.M)

						// fix for issue where Array of type strings is not properly
						// allowing filter with $in keyword
						if bsonType, ok := value["bsonType"]; ok {
							bsonType := bsonType.(string)
							// capture type in the fieldTypes map
							if bsonType != "" {
								qb.fieldTypes[fmt.Sprintf("%s%s", parentPrefix, field)] = bsonType
							}
						}
					}
				}

				// handle any sub-document schema details
				if subProperties, ok := value["properties"]; ok {
					subProperties := subProperties.(bson.M)
					qb.iterateProperties(
						fmt.Sprintf("%s%s.", parentPrefix, field), subProperties)
				}

				continue
			}

			// check for enum (without bsonType specified)
			if _, ok := value["enum"]; ok {
				qb.fieldTypes[fmt.Sprintf("%s%s", parentPrefix, field)] = "object"
			}
		default:
			// properties are not of type bson.M
			continue
		}
	}
}

func (qb QueryBuilder) setPaginationOptions(pagination map[string]int, opts *options.FindOptions) {
	// check for limit
	if limit, ok := pagination["limit"]; ok {
		opts.SetLimit(int64(limit))

		// check for offset (once limit is set)
		if offset, ok := pagination["offset"]; ok {
			opts.SetSkip(int64(offset))
		}

		// check for skip (once limit is set)
		if skip, ok := pagination["skip"]; ok {
			opts.SetSkip(int64(skip))
		}
	}

	// check for page and size
	if size, ok := pagination["size"]; ok {
		opts.SetLimit(int64(size))

		// set skip (requires understanding of size)
		if page, ok := pagination["page"]; ok {
			opts.SetSkip(int64(page * size))
		}
	}
}

func (qb QueryBuilder) setProjectionOptions(fields []string, opts *options.FindOptions) error {
	// set field projections option
	if len(fields) > 0 {
		prj := map[string]int{}
		for _, field := range fields {
			val := 1

			// handle when the first char is a - (don't display field in result)
			if field[0:1] == "-" {
				field = field[1:]
				val = 0
			}

			// handle scenarios where the first char is a + (redundant)
			if field[0:1] == "+" {
				field = field[1:]
			}

			// lookup field in the fieldTypes dictionary if strictValidation is true
			if qb.strictValidation {
				field = strings.Split(field, "[]")[0]
				if _, ok := qb.fieldTypes[field]; !ok {
					// we have a problem
					return fmt.Errorf("field %s does not exist in collection %s", field, qb.collection)
				}
				field = strings.ReplaceAll(field, "[]", ".")
			}

			// add the field to the project dictionary
			prj[field] = val
		}

		// add the projection to the FindOptions
		if len(prj) > 0 {
			opts.SetProjection(prj)
		}
	}

	return nil
}

func (qb QueryBuilder) setSortOptions(fields []string, opts *options.FindOptions) error {
	if len(fields) > 0 {
		sort := bson.D{}
		for _, field := range fields {
			val := 1

			if field[0:1] == "-" {
				field = field[1:]
				val = -1
			}

			if field[0:1] == "+" {
				field = field[1:]
			}

			// lookup field in the fieldTypes dictionary if strictValidation is true
			if qb.strictValidation {
				fiendNameWithNoIdx := strings.Split(field, "[]")[0]
				if _, ok := qb.fieldTypes[fiendNameWithNoIdx]; !ok {
					// we have a problem
					return fmt.Errorf("field %s does not exist in collection %s", fiendNameWithNoIdx, qb.collection)
				}
				field = strings.ReplaceAll(field, "[]", ".")
			}

			sort = append(sort, bson.E{Key: field, Value: val})
		}

		opts.SetSort(sort)
	}

	return nil
}
