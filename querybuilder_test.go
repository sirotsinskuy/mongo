package querybuilder

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	queryoptions "go.jtlabs.io/query"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	options "go.mongodb.org/mongo-driver/mongo/options"
)

func Test_NewQueryBuilder(t *testing.T) {
	type args struct {
		collection       string
		schema           bson.M
		strictValidation []bool
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "test with strict validation specified",
			args: args{
				collection:       "test",
				schema:           bson.M{},
				strictValidation: []bool{true},
			},
			want: map[string]string{},
		},
		{
			name: "test with basic schema",
			args: args{
				collection: "test",
				schema: bson.M{
					"bsonType": "object",
					"required": []string{"someID", "created", "someName"},
					"properties": bson.M{
						"someID": bson.M{
							"bsonType":    "string",
							"description": "primary identifier of something, must be unique",
						},
						"created": bson.M{
							"bsonType":    "date",
							"description": "date for when the thing was created",
						},
						"someName": bson.M{
							"bsonType":    "string",
							"description": "string name of the thing",
						},
						"disabled": bson.M{
							"bsonType":    "bool",
							"description": "boolean type",
						},
						"customEnum": bson.M{
							"enum":        bson.A{"A", "B", "C"},
							"description": "an enum type",
						},
						"minMaxNumber": bson.M{
							"bsonType":    "int",
							"minimum":     0,
							"maximum":     100,
							"description": "number with a min and max",
						},
						"childStructureNoSchema": bson.M{
							"bsonType":    "object",
							"description": "child structure with no schema",
						},
						"childArray": bson.M{
							"bsonType": "array",
							"items": bson.M{
								"bsonType": "object",
								"properties": bson.M{
									"field1": bson.M{
										"bsonType":    "string",
										"description": "sub document in array field 1",
									},
									"field2": bson.M{
										"bsonType":    "string",
										"description": "sub document in array field 2",
									},
								},
							},
						},
						"childStringArray": bson.M{
							"bsonType": "array",
							"items": bson.M{
								"bsonType": "string",
							},
						},
						"childStructure": bson.M{
							"bsonType": "object",
							"required": bson.A{},
							"properties": bson.M{
								"fieldA": bson.M{
									"bsonType":    "array",
									"description": "an array of elements",
								},
								"fieldB": bson.M{
									"bsonType":    "date",
									"description": "a nested date value",
								},
								"fieldC": bson.M{
									"bsonType": "object",
									"required": bson.A{"fieldC-1"},
									"properties": bson.M{
										"fieldC-1": bson.M{
											"bsonType":    "string",
											"description": "nested two layers deep string",
										},
										"fieldC-2": bson.M{
											"bsonType":    "double",
											"description": "a double value",
										},
									},
								},
							},
						},
						"notAMap": bson.D{{
							Key:   "notAMap",
							Value: "for testing purposes",
						}},
					},
				},
			},
			want: map[string]string{
				"someID":                         "string",
				"created":                        "date",
				"someName":                       "string",
				"disabled":                       "bool",
				"minMaxNumber":                   "int",
				"childArray":                     "object",
				"childArray.field1":              "string",
				"childArray.field2":              "string",
				"childStringArray":               "string",
				"childStructureNoSchema":         "object",
				"childStructure":                 "object",
				"childStructure.fieldB":          "date",
				"childStructure.fieldC":          "object",
				"childStructure.fieldC.fieldC-1": "string",
				"childStructure.fieldC.fieldC-2": "double",
				"childStructure.fieldA":          "array",
				"customEnum":                     "object",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var qb *QueryBuilder
			if len(tt.args.strictValidation) > 0 {
				qb = NewQueryBuilder(tt.args.collection, tt.args.schema, tt.args.strictValidation...)
			} else {
				qb = NewQueryBuilder(tt.args.collection, tt.args.schema)
			}

			if !reflect.DeepEqual(qb.fieldTypes, tt.want) {
				t.Errorf("NewQueryBuilder(), qb.fieldTypes = %v, want %v", qb.fieldTypes, tt.want)
			}

			if len(tt.args.strictValidation) > 0 {
				sv := tt.args.strictValidation[0]
				if sv != qb.strictValidation {
					t.Errorf("NewQueryBuilder(), qb.strictValidation = %v, want %v", qb.strictValidation, sv)
				}
			}
		})
	}
}

func TestQueryBuilder_Filter(t *testing.T) {
	type fields struct {
		collection       string
		fieldTypes       map[string]string
		strictValidation bool
	}
	type args struct {
		qs string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bson.M
		wantErr bool
	}{
		{
			name: "test with no query args",
			fields: fields{
				collection:       "test",
				fieldTypes:       map[string]string{},
				strictValidation: false,
			},
			args: args{
				qs: "",
			},
			want:    bson.M{},
			wantErr: false,
		},
		{
			name: "should error with strict validation and mismatched field",
			fields: fields{
				collection:       "test",
				fieldTypes:       map[string]string{},
				strictValidation: true,
			},
			args: args{
				qs: "filter[nofield]=error",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "should not error without strict validation and mismatched field",
			fields: fields{
				collection:       "test",
				fieldTypes:       map[string]string{},
				strictValidation: false,
			},
			args: args{
				qs: "filter[nofield]=error",
			},
			want:    bson.M{},
			wantErr: false,
		},
		{
			name: "should properly detect and type numeric values",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"deVal": "decimal",
					"doVal": "double",
					"iVal":  "int",
					"lVal":  "long",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[doVal]=0.000000000000000000000000000000009&filter[deVal]=10.01&filter[iVal]=2147483647&filter[lVal]=9223372036854775807",
			},
			want: bson.M{
				"deVal": float32(10.01),
				"doVal": float64(0.000000000000000000000000000000009),
				"iVal":  int32(2147483647),
				"lVal":  int64(9223372036854775807),
			},
			wantErr: false,
		},
		{
			name: "should properly handle numeric values with $in operator",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"iVal1": "int",
					"iVal2": "decimal",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[iVal1]=1,2,3,4,5&filter[iVal2]=1.1,2.2,3.3",
			},
			want: bson.M{
				"iVal1": bson.D{primitive.E{
					Key:   "$in",
					Value: primitive.A{int32(1), int32(2), int32(3), int32(4), int32(5)},
				}},
				"iVal2": bson.D{primitive.E{
					Key:   "$in",
					Value: primitive.A{float32(1.1), float32(2.2), float32(3.3)},
				}},
			},
			wantErr: false,
		},
		{
			name: "should properly handle numeric values with $all operator",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"iVal1": "int",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[iVal1]={}0,1,2,3,4,5",
			},
			want: bson.M{
				"iVal1": bson.D{primitive.E{
					Key:   "$all",
					Value: primitive.A{int32(0), int32(1), int32(2), int32(3), int32(4), int32(5)},
				}},
			},
			wantErr: false,
		},
		{
			name: "should properly handle $elemMatch operator using []",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"aVal":       "array",
					"aVal.iVal1": "int",
					"aVal.iVal2": "date",
					"aVal.iVal3": "int",
				},
			},
			args: args{
				qs: "filter[aVal.iVal1]=[]1&filter[aVal.iVal2]=[]nil",
			},
			want: bson.M{
				"aVal": bson.M{
					"$elemMatch": bson.M{
						"iVal1": 1,
						"iVal2": nil,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should properly handle $elemMatch operator using [] for complex arrays with Dates",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"aVal":                  "array",
					"aVal.innerArray":       "array",
					"aVal.innerArray.iVal1": "date",
				},
			},
			args: args{
				qs: "filter[aVal.[*].innerArray.iVal1]=-2020-01-01T12:00:00.000Z,-2020-01-02T12:00:00.000Z",
			},
			want: bson.M{
				"aVal": bson.M{
					"$elemMatch": bson.M{
						"$or": bson.A{
							bson.M{
								"innerArray.iVal1": bson.M{
									"$nin": bson.A{
										time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
										time.Date(2020, time.January, 2, 12, 0, 0, 0, time.UTC),
									},
								},
							},
							bson.M{
								"innerArray.iVal1": nil,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should properly handle $or operator for regexp",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"iVal1": "string",
					"iVal2": "string",
					"iVal3": "string",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[iVal1]=||*test*&filter[iVal2]=||*test2*&filter[iVal3]=test3",
			},
			want: bson.M{
				"$or": bson.A{
					bson.M{
						"iVal1": primitive.Regex{Pattern: "test", Options: "im"},
					},
					bson.M{
						"iVal2": primitive.Regex{Pattern: "test2", Options: "im"},
					},
				},
				"iVal3": "test3",
			},
			wantErr: false,
		},
		/*TODO fixme: {
			name: "should properly handle $or operator for date and number",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"iVal1": "date",
					"iVal2": "int",
					"iVal3": "string",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[iVal1]=||>2020-01-01T12:00:00.000Z&filter[iVal2]=||100&filter[iVal3]=test3",
			},
			want: bson.M{
				"$or": bson.A{
					bson.M{
						"iVal1": bson.E{
							Key:   "$gt",
							Value: time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
						},
					},
					bson.M{
						"iVal2": int32(100),
					},
				},
				"iVal3": "test3",
			},
			wantErr: false,
		},*/
		{
			name: "should properly handle $or operator for date and range of numbers",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"iVal1": "date",
					"iVal2": "int",
					"iVal3": "string",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[iVal1]=||>2020-01-01T12:00:00.000Z&filter[iVal2]=||!=100&filter[iVal3]=test3",
			},
			want: bson.M{
				"$or": bson.A{
					bson.M{
						"iVal1": bson.E{
							Key:   "$gt",
							Value: time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
						},
					},
					bson.M{
						"iVal2": bson.E{
							Key:   "$ne",
							Value: int32(100),
						},
					},
				},
				"iVal3": "test3",
			},
			wantErr: false,
		},
		{
			name: "should properly handle numeric values with range (><) operator",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"iVal1": "int",
					"iVal2": "decimal",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[iVal1]=><1,4&filter[iVal2]=><1.1,2.2",
			},
			want: bson.M{
				"iVal1": bson.D{
					primitive.E{
						Key:   "$gte",
						Value: int32(1),
					},
					primitive.E{
						Key:   "$lte",
						Value: int32(4),
					},
				},
				"iVal2": bson.D{
					primitive.E{
						Key:   "$gte",
						Value: float32(1.1),
					},
					primitive.E{
						Key:   "$lte",
						Value: float32(2.2),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should properly handle numeric operators (lt, lte, gt, gte, ne)",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"iVal1": "int",
					"iVal2": "int",
					"iVal3": "int",
					"iVal4": "int",
					"iVal5": "int",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[iVal1]=%3C4&filter[iVal2]=%3C%3D3&filter[iVal3]=%3E1&filter[iVal4]=%3E%3D2&filter[iVal5]=%21%3D5",
			},
			want: bson.M{
				"iVal1": bson.D{primitive.E{
					Key:   "$lt",
					Value: int32(4),
				}},
				"iVal2": bson.D{primitive.E{
					Key:   "$lte",
					Value: int32(3),
				}},
				"iVal3": bson.D{primitive.E{
					Key:   "$gt",
					Value: int32(1),
				}},
				"iVal4": bson.D{primitive.E{
					Key:   "$gte",
					Value: int32(2),
				}},
				"iVal5": bson.D{primitive.E{
					Key:   "$ne",
					Value: int32(5),
				}},
			},
			wantErr: false,
		},
		{
			name: "should properly handle numeric operators (lt, lte, gt, gte, ne) with array",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"iVal1": "int",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[iVal1[]1]=%3C4",
			},
			want: bson.M{
				"iVal1.1": bson.D{primitive.E{
					Key:   "$lt",
					Value: int32(4),
				}},
			},
			wantErr: false,
		},
		{
			name: "should properly handle bool types",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"bVal1": "bool",
					"bVal2": "bool",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[bVal1]=true&filter[bVal2]=false",
			},
			want: bson.M{
				"bVal1": true,
				"bVal2": false,
			},
			wantErr: false,
		},
		{
			name: "should properly handle simple numeric",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"bVal1": "int",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[bVal1]=1",
			},
			want: bson.M{
				"bVal1": int32(1),
			},
			wantErr: false,
		},
		{
			name: "should properly handle bool types with $ne",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"bVal1": "bool",
					"bVal2": "bool",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[bVal1]=true&filter[bVal2]=-false",
			},
			want: bson.M{
				"bVal1": true,
				"bVal2": primitive.M{
					"$ne": false,
				}},
			wantErr: false,
		},
		{
			name: "should properly handle date types",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"dVal1": "date",
					"dVal2": "date",
					"dVal3": "date",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[dVal1]=2020-01-01T12:00:00.000Z&filter[dVal2]=2021-02-16T02:04:05.000Z&filter[dVal3]=2021-02-16T02:04:05.000Z,2020-01-01T12:00:00.000Z",
			},
			want: bson.M{
				"dVal1": time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
				"dVal2": time.Date(2021, time.February, 16, 2, 4, 5, 0, time.UTC),
				"dVal3": bson.D{primitive.E{
					Key:   "$in",
					Value: primitive.A{time.Date(2021, time.February, 16, 2, 4, 5, 0, time.UTC), time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC)},
				}},
			},
			wantErr: false,
		},
		{
			name: "should properly handle operators on date types",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"dVal1": "date",
					"dVal2": "date",
					"dVal3": "date",
					"dVal4": "date",
					"dVal5": "date",
					"dVal6": "date",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[dVal1]=<2020-01-01T12:00:00.000Z&filter[dVal2]=<=2021-02-16T02:04:05.000Z&filter[dVal3]=>2021-02-16T02:04:05.000Z&filter[dVal4]=>=2021-02-16T02:04:05.000Z&filter[dVal5]=!=2020-01-01T12:00:00.000Z&filter[dVal6]=-2020-01-01T12:00:00.000Z",
			},
			want: bson.M{
				"dVal1": bson.D{primitive.E{
					Key:   "$lt",
					Value: time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
				}},
				"dVal2": bson.D{primitive.E{
					Key:   "$lte",
					Value: time.Date(2021, time.February, 16, 2, 4, 5, 0, time.UTC),
				}},
				"dVal3": bson.D{primitive.E{
					Key:   "$gt",
					Value: time.Date(2021, time.February, 16, 2, 4, 5, 0, time.UTC),
				}},
				"dVal4": bson.D{primitive.E{
					Key:   "$gte",
					Value: time.Date(2021, time.February, 16, 2, 4, 5, 0, time.UTC),
				}},
				"dVal5": bson.D{primitive.E{
					Key:   "$ne",
					Value: time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
				}},
				"dVal6": bson.D{primitive.E{
					Key:   "$ne",
					Value: time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
				}},
			},
			wantErr: false,
		},
		{
			name: "should properly handle range operator on date types",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"dVal1": "date",
					"dVal2": "date",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[dVal1]=><2020-01-01T12:00:00.000Z,2021-02-16T02:04:05.000Z&filter[dVal2]=><2021-02-16T02:04:05.000Z,2021-02-16T02:04:05.000Z",
			},
			want: bson.M{
				"dVal1": bson.D{
					primitive.E{
						Key:   "$gte",
						Value: time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
					},
					primitive.E{
						Key:   "$lte",
						Value: time.Date(2021, time.February, 16, 2, 4, 5, 0, time.UTC),
					},
				},
				"dVal2": bson.D{
					primitive.E{
						Key:   "$gte",
						Value: time.Date(2021, time.February, 16, 2, 4, 5, 0, time.UTC),
					},
					primitive.E{
						Key:   "$lte",
						Value: time.Date(2021, time.February, 16, 2, 4, 5, 0, time.UTC),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should properly handle timestamp types",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"dVal1": "timestamp",
					"dVal2": "timestamp",
					"dVal3": "timestamp",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[dVal1]=2020-01-01T12:00:00.000Z&filter[dVal2]=2021-02-16T02:04:05.000Z&filter[dVal3]=2021-02-16T02:04:05.000Z,2020-01-01T12:00:00.000Z",
			},
			want: bson.M{
				"dVal1": time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
				"dVal2": time.Date(2021, time.February, 16, 2, 4, 5, 0, time.UTC),
				"dVal3": bson.D{primitive.E{
					Key:   "$in",
					Value: primitive.A{time.Date(2021, time.February, 16, 2, 4, 5, 0, time.UTC), time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC)},
				}},
			},
			wantErr: false,
		},
		{
			name: "should properly handle string type using $exists operator with object fields",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"oVal":       "object",
					"oVal.sVal1": "string",
					"oVal.sVal2": "string",
					"oVal.sVal3": "string",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[oVal]=sVal1,!=sVal2,-sVal3",
			},
			want: bson.M{
				"oVal.sVal1": bson.D{primitive.E{
					Key:   "$exists",
					Value: true,
				}},
				"oVal.sVal2": bson.D{primitive.E{
					Key:   "$exists",
					Value: false,
				}},
				"oVal.sVal3": bson.D{primitive.E{
					Key:   "$exists",
					Value: false,
				}},
			},
			wantErr: false,
		},
		{
			name: "should properly handle string type using $in operator with array of values",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"sVal1": "string",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[sVal1]=value1,value2,value3",
			},
			want: bson.M{
				"sVal1": bson.D{primitive.E{
					Key:   "$in",
					Value: primitive.A{"value1", "value2", "value3"},
				}},
			},
			wantErr: false,
		},
		{
			name: "should properly handle date type using $nin operator with array of values",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"sVal1": "date",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[sVal1]=-2020-01-01T12:00:00.000Z,-2020-01-02T12:00:00.000Z",
			},
			want: bson.M{
				"sVal1": bson.D{primitive.E{
					Key: "$nin",
					Value: primitive.A{
						time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
						time.Date(2020, time.January, 2, 12, 0, 0, 0, time.UTC)},
				}},
			},
			wantErr: false,
		},
		{
			name: "should properly handle array type and not use $in operator with array of values",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"aVal1": "array",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[aVal1]=value1,value2,value3",
			},
			want: bson.M{
				"aVal1": primitive.A{"value1", "value2", "value3"},
			},
			wantErr: false,
		},
		{
			name: "should properly handle string wildcards with regexes",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"sVal1": "string",
					"sVal2": "string",
					"sVal3": "string",
					"sVal4": "string",
					"sVal5": "string",
					"sVal6": "string",
					"sVal7": "string",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[sVal1]=*value&filter[sVal2]=value*&filter[sVal3]=*value*&filter[sVal4]=value&filter[sVal5]=!=value&filter[sVal6]=\"value\"&filter[sVal7]=-value",
			},
			want: bson.M{
				"sVal1": primitive.Regex{
					Pattern: "value$",
					Options: "im",
				},
				"sVal2": primitive.Regex{
					Pattern: "^value",
					Options: "im",
				},
				"sVal3": primitive.Regex{
					Pattern: "value",
					Options: "im",
				},
				"sVal4": "value",
				"sVal5": bson.D{primitive.E{
					Key:   "$ne",
					Value: "value",
				}},
				"sVal6": primitive.Regex{
					Pattern: "^value$",
					Options: "",
				},
				"sVal7": bson.D{primitive.E{
					Key:   "$ne",
					Value: "value",
				}},
			},
			wantErr: false,
		},
		{
			name: "should properly handle null keyword in searches",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"sVal1": "string",
					"nVal1": "int",
					"dVal1": "date",
					"sVal2": "string",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[sVal1]=null&filter[nVal1]=-null&filter[dVal1]=null&filter[sVal2]=-null",
			},
			want: bson.M{
				"sVal1": nil,
				"nVal1": bson.D{primitive.E{
					Key:   "$ne",
					Value: nil,
				}},
				"dVal1": nil,
				"sVal2": bson.D{primitive.E{
					Key:   "$ne",
					Value: nil,
				}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := QueryBuilder{
				collection:       tt.fields.collection,
				fieldTypes:       tt.fields.fieldTypes,
				strictValidation: tt.fields.strictValidation,
			}

			qo, err := queryoptions.FromQuerystring(tt.args.qs)
			if err != nil {
				t.Errorf("options.FromQuerystring() error = %v", err)
				return
			}

			got, err := qb.Filter(qo)
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryBuilder.Filter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// check to see if it matches expectations
			if !reflect.DeepEqual(got, tt.want) && fmt.Sprintf("%+v", got) != fmt.Sprintf("%+v", tt.want) {
				// values do not match
				t.Errorf("QueryBuilder.Filter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueryBuilder_FindOptions(t *testing.T) {
	var el int64 = 100

	type fields struct {
		collection       string
		fieldTypes       map[string]string
		strictValidation bool
	}
	type args struct {
		qo queryoptions.Options
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *options.FindOptions
		wantErr bool
	}{
		{
			name: "should properly determine Limit options with query options defined limit",
			fields: fields{
				collection:       "test",
				fieldTypes:       map[string]string{},
				strictValidation: false,
			},
			args: args{
				qo: queryoptions.Options{
					Page: map[string]int{
						"limit": 100,
					},
				},
			},
			want: &options.FindOptions{
				Limit: &el,
			},
			wantErr: false,
		},
		{
			name: "should properly determine Limit options with query options defined size",
			fields: fields{
				collection:       "test",
				fieldTypes:       map[string]string{},
				strictValidation: false,
			},
			args: args{
				qo: queryoptions.Options{
					Page: map[string]int{
						"size": 100,
					},
				},
			},
			want: &options.FindOptions{
				Limit: &el,
			},
			wantErr: false,
		},
		{
			name: "should properly determine Skip options with query options defined limit and offset",
			fields: fields{
				collection:       "test",
				fieldTypes:       map[string]string{},
				strictValidation: false,
			},
			args: args{
				qo: queryoptions.Options{
					Page: map[string]int{
						"limit":  100,
						"offset": 100,
					},
				},
			},
			want: &options.FindOptions{
				Limit: &el,
				Skip:  &el,
			},
			wantErr: false,
		},
		{
			name: "should properly determine Skip options with query options defined limit and skip",
			fields: fields{
				collection:       "test",
				fieldTypes:       map[string]string{},
				strictValidation: false,
			},
			args: args{
				qo: queryoptions.Options{
					Page: map[string]int{
						"limit": 100,
						"skip":  100,
					},
				},
			},
			want: &options.FindOptions{
				Limit: &el,
				Skip:  &el,
			},
			wantErr: false,
		},
		{
			name: "should properly determine Skip and Size options with query options defined page and size",
			fields: fields{
				collection:       "test",
				fieldTypes:       map[string]string{},
				strictValidation: false,
			},
			args: args{
				qo: queryoptions.Options{
					Page: map[string]int{
						"page": 1,
						"size": 100,
					},
				},
			},
			want: &options.FindOptions{
				Limit: &el,
				Skip:  &el,
			},
			wantErr: false,
		},
		{
			name: "should properly determine projection fields when provided",
			fields: fields{
				collection:       "test",
				fieldTypes:       map[string]string{},
				strictValidation: false,
			},
			args: args{
				// notice use of + before fieldC to validate + prefix to field name
				qo: queryoptions.Options{
					Fields: []string{"fieldA", "fieldB", "+fieldC"},
				},
			},
			want: &options.FindOptions{
				Projection: map[string]int{
					"fieldA": 1,
					"fieldB": 1,
					"fieldC": 1,
				},
			},
			wantErr: false,
		},
		{
			name: "should properly determine excluded fields in projection when provided",
			fields: fields{
				collection:       "test",
				fieldTypes:       map[string]string{},
				strictValidation: false,
			},
			args: args{
				qo: queryoptions.Options{
					Fields: []string{"-fieldA"},
				},
			},
			want: &options.FindOptions{
				Projection: map[string]int{
					"fieldA": 0,
				},
			},
			wantErr: false,
		},
		{
			name: "should properly error when providing a field in projection that does not exist and strict validation is true",
			fields: fields{
				collection:       "test",
				fieldTypes:       map[string]string{},
				strictValidation: true,
			},
			args: args{
				qo: queryoptions.Options{
					Fields: []string{"-fieldA"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		/*TODO fixme:
		{
			name: "should properly sort when sort details are provided",
			fields: fields{
				collection:       "test",
				fieldTypes:       map[string]string{},
				strictValidation: false,
			},
			args: args{
				// notice the use of + and - as field prefixes below
				qo: queryoptions.Options{
					Sort: []string{"fieldA", "+fieldB", "-fieldC"},
				},
			},
			want: &options.FindOptions{
				Sort: map[string]int{
					"fieldA": 1,
					"fieldB": 1,
					"fieldC": -1,
				},
			},
			wantErr: false,
		},
		{
			name: "should properly error when providing a field in sort that does not exist and strict validation is true",
			fields: fields{
				collection:       "test",
				fieldTypes:       map[string]string{},
				strictValidation: true,
			},
			args: args{
				qo: queryoptions.Options{
					Sort: []string{"-fieldA"},
				},
			},
			want:    nil,
			wantErr: true,
		},*/
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := QueryBuilder{
				collection:       tt.fields.collection,
				fieldTypes:       tt.fields.fieldTypes,
				strictValidation: tt.fields.strictValidation,
			}
			got, err := qb.FindOptions(tt.args.qo)

			if (err != nil) != tt.wantErr {
				t.Errorf("QueryBuilder.FindOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("QueryBuilder.FindOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}
