// Copyright 2021 FerretDB Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package query

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/FerretDB/FerretDB/v2/integration"
	"github.com/FerretDB/FerretDB/v2/integration/setup"
	"github.com/FerretDB/FerretDB/v2/integration/shareddata"
)

func TestFindAndModifyCompatSimple(t *testing.T) {
	t.Parallel()

	testCases := map[string]findAndModifyCompatTestCase{
		"EmptyQueryRemove": {
			command: bson.D{
				{"query", bson.D{}},
				{"remove", true},
			},
		},
		"NewDoubleNonZero": {
			command: bson.D{
				{"query", bson.D{{"_id", "double-smallest"}}},
				{"update", bson.D{{"_id", "double-smallest"}, {"v", float64(43)}}},
				{"new", float64(42)},
			},
		},
		"NewDoubleZero": {
			command: bson.D{
				{"query", bson.D{{"_id", "double-zero"}}},
				{"update", bson.D{{"_id", "double-zero"}, {"v", 43.0}}},
				{"new", float64(0)},
			},
		},
		"NewIntNonZero": {
			command: bson.D{
				{"query", bson.D{{"_id", "int32"}}},
				{"update", bson.D{{"_id", "int32"}, {"v", int32(43)}}},
				{"new", int32(11)},
			},
		},
		"NewIntZero": {
			command: bson.D{
				{"query", bson.D{{"_id", "int32-zero"}}},
				{"update", bson.D{{"_id", "int32-zero"}, {"v", int32(43)}}},
				{"new", int32(0)},
			},
		},
		"NewLongNonZero": {
			command: bson.D{
				{"query", bson.D{{"_id", "int64"}}},
				{"update", bson.D{{"_id", "int64"}, {"v", int64(43)}}},
				{"new", int64(11)},
			},
		},
		"NewLongZero": {
			command: bson.D{
				{"query", bson.D{{"_id", "int64-zero"}}},
				{"update", bson.D{{"_id", "int64-zero"}, {"v", int64(43)}}},
				{"new", int64(0)},
			},
		},
	}

	testFindAndModifyCompat(t, testCases)
}

func TestFindAndModifyCompatErrors(t *testing.T) {
	t.Parallel()

	testCases := map[string]findAndModifyCompatTestCase{
		"NotEnoughParameters": {
			command:    bson.D{},
			resultType: integration.EmptyResult,
		},
		"UpdateAndRemove": {
			command: bson.D{
				{"update", bson.D{}},
				{"remove", true},
			},
			resultType: integration.EmptyResult,
		},
		"NewAndRemove": {
			command: bson.D{
				{"new", true},
				{"remove", true},
			},
			resultType: integration.EmptyResult,
		},
		"InvalidUpdateType": {
			command: bson.D{
				{"query", bson.D{}},
				{"update", "123"},
			},
			resultType: integration.EmptyResult,
		},
		"InvalidMaxTimeMSType": {
			command: bson.D{
				{"maxTimeMS", "string"},
			},
			resultType:       integration.EmptyResult,
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/318",
		},
		"DuplicateID": {
			command: bson.D{
				{"query", bson.D{{"non-existent", "val"}}},
				{"update", bson.D{{"_id", "int32"}, {"v", int32(43)}}},
				{"upsert", true},
			},
			providers:        []shareddata.Provider{shareddata.Int32s},
			resultType:       integration.EmptyResult,
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/894",
		},
		"InvalidID": {
			command: bson.D{
				{"query", bson.D{{"non-existent", "val"}}},
				{"update", bson.D{{"_id", primitive.Regex{Pattern: "[a-z]*[0-9]"}}, {"v", int32(43)}}},
				{"upsert", true},
			},
			providers:        []shareddata.Provider{shareddata.Int32s},
			resultType:       integration.EmptyResult,
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/317",
		},
	}

	testFindAndModifyCompat(t, testCases)
}

func TestFindAndModifyCompatUpdate(t *testing.T) {
	t.Parallel()

	testCases := map[string]findAndModifyCompatTestCase{
		"Replace": {
			command: bson.D{
				{"query", bson.D{{"_id", "int64"}}},
				{"update", bson.D{{"_id", "int64"}, {"v", int64(43)}}},
			},
		},
		"ReplaceWithoutID": {
			command: bson.D{
				{"query", bson.D{{"_id", "int64"}}},
				{"update", bson.D{{"v", int64(43)}}},
			},
		},
		"ReplaceReturnNew": {
			command: bson.D{
				{"query", bson.D{{"_id", "int32"}}},
				{"update", bson.D{{"_id", "int32"}, {"v", int32(43)}}},
				{"new", true},
			},
		},
		"NotExistedIdInQuery": {
			command: bson.D{
				{"query", bson.D{{"_id", "no-such-id"}}},
				{"update", bson.D{{"v", int32(43)}}},
			},
		},
		"NotExistedIdNotInQuery": {
			command: bson.D{
				{"query", bson.D{{"$and", bson.A{
					bson.D{{"v", bson.D{{"$gt", 0}}}},
					bson.D{{"v", bson.D{{"$lt", 0}}}},
				}}}},
				{"update", bson.D{{"v", int32(43)}}},
			},
		},
		"UpdateOperatorSet": {
			command: bson.D{
				{"query", bson.D{{"_id", "int64"}}},
				{"update", bson.D{{"$set", bson.D{{"v", int64(43)}}}}},
			},
		},
		"UpdateOperatorSetReturnNew": {
			command: bson.D{
				{"query", bson.D{{"_id", "int64"}}},
				{"update", bson.D{{"$set", bson.D{{"v", int64(43)}}}}},
				{"new", true},
			},
		},
		"EmptyUpdate": {
			command: bson.D{
				{"query", bson.D{{"_id", "int64"}}},
				{"update", bson.D{{"v", bson.D{}}}},
			},
		},
		"Conflict": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", false}}}}},
				{"update", bson.D{{"$invalid", "non-existent-field"}}},
			},
			resultType: integration.EmptyResult,
		},
		"OperatorConflict": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", false}}}}},
				{"update", bson.D{
					{"$set", bson.D{{"v", 4}}},
					{"$inc", bson.D{{"v", 4}}},
				}},
			},
			resultType: integration.EmptyResult,
		},
		"NoConflict": {
			command: bson.D{
				{"query", bson.D{{"_id", "int64"}}},
				{"update", bson.D{
					{"$set", bson.D{{"v", 4}}},
					{"$inc", bson.D{{"foo", 4}}},
				}},
			},
		},
		"EmptyKey": {
			command: bson.D{
				{"query", bson.D{{"_id", "int64"}}},
				{"update", bson.D{
					{"$set", bson.D{{"", 4}}},
					{"$inc", bson.D{{"", 4}}},
				}},
			},
			resultType: integration.EmptyResult,
		},
		"EmptyKeyAndKey": {
			command: bson.D{
				{"query", bson.D{{"_id", "int64"}}},
				{"update", bson.D{
					{"$set", bson.D{{"", 4}}},
					{"$inc", bson.D{{"v", 4}}},
				}},
			},
			resultType: integration.EmptyResult,
		},
		"InvalidOperator": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", false}}}}},
				{"update", bson.D{{"$invalid", "non-existent-field"}}},
			},
			resultType: integration.EmptyResult,
		},
	}

	testFindAndModifyCompat(t, testCases)
}

func TestFindAndModifyCompatDotNotation(t *testing.T) {
	testCases := map[string]findAndModifyCompatTestCase{
		"Conflict": {
			command: bson.D{
				{"query", bson.D{{"_id", "array-documents-two-fields"}}},
				{"update", bson.D{
					{"$set", bson.D{{"v.0.field", 4}}},
					{"$inc", bson.D{{"v.0.field", 4}}},
				}},
			},
			resultType: integration.EmptyResult,
		},
		"NoConflict": {
			command: bson.D{
				{"query", bson.D{{"_id", "array-documents-two-fields"}}},
				{"update", bson.D{
					{"$set", bson.D{{"v.0.field", 4}}},
					{"$inc", bson.D{{"v.0.foo", 4}}},
				}},
			},
		},
		"NoIndex": {
			command: bson.D{
				{"query", bson.D{{"_id", "array-documents-two-fields"}}},
				{"update", bson.D{
					{"$set", bson.D{{"v.0.field", 4}}},
					{"$inc", bson.D{{"v.field", 4}}},
				}},
			},
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/320",
			failsProviders:   []shareddata.Provider{shareddata.ArrayAndDocuments},
		},
		"ParentConflict": {
			command: bson.D{
				{"query", bson.D{{"_id", "array-documents-two-fields"}}},
				{"update", bson.D{
					{"$set", bson.D{{"v.0.field", 4}}},
					{"$inc", bson.D{{"v", 4}}},
				}},
			},
			resultType: integration.EmptyResult,
		},

		"ConflictKey": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", false}}}}},
				{"update", bson.D{
					{"$set", bson.D{{"v", "val"}}},
					{"$min", bson.D{{"v.foo", "val"}}},
				}},
			},
			resultType: integration.EmptyResult,
		},
		"ConflictKeyPrefix": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", false}}}}},
				{"update", bson.D{
					{"$set", bson.D{{"v.foo", "val"}}},
					{"$min", bson.D{{"v", "val"}}},
				}},
			},
			resultType: integration.EmptyResult,
		},
	}

	testFindAndModifyCompat(t, testCases)
}

func TestFindAndModifyCompatUpdateSet(t *testing.T) {
	t.Parallel()

	testCases := map[string]findAndModifyCompatTestCase{
		"NonExistentExistsTrue": {
			command: bson.D{
				{"query", bson.D{{"non-existent", bson.D{{"$exists", true}}}}},
				{"update", bson.D{{"$set", bson.D{{"v", "foo"}}}}},
			},
		},
		"NonExistentExistsFalse": {
			command: bson.D{
				{"query", bson.D{{"non-existent", bson.D{{"$exists", false}}}}},
				{"update", bson.D{{"$set", bson.D{{"v", "foo"}}}}},
			},
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/321",
			failsProviders:   []shareddata.Provider{shareddata.Strings},
		},
		"ExistsTrue": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", true}}}}},
				{"update", bson.D{{"$set", bson.D{{"v", "foo"}}}}},
			},
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/321",
			failsProviders:   []shareddata.Provider{shareddata.Strings},
		},
		"ExistsFalse": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", false}}}}},
				{"update", bson.D{{"$set", bson.D{{"v", "foo"}}}}},
			},
		},
		"UpdateIDNoQuery": {
			command: bson.D{
				{"update", bson.D{{"$set", bson.D{{"_id", "int32"}}}}},
			},
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/321",
			failsProviders:   []shareddata.Provider{shareddata.Int32s},
		},
		"UpdateExistingID": {
			command: bson.D{
				{"query", bson.D{{"_id", "int32"}}},
				{"update", bson.D{{"$set", bson.D{{"_id", "int32-1"}}}}},
			},
		},
		"UpdateSameID": {
			command: bson.D{
				{"query", bson.D{{"_id", "int32"}}},
				{"update", bson.D{{"$set", bson.D{{"_id", "int32"}}}}},
			},
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/321",
			failsProviders:   []shareddata.Provider{shareddata.Int32s, shareddata.Scalars},
		},
	}

	testFindAndModifyCompat(t, testCases)
}

func TestFindAndModifyCompatUnset(t *testing.T) {
	t.Parallel()

	testCases := map[string]findAndModifyCompatTestCase{
		"NonExistentExistsT": {
			command: bson.D{
				{"query", bson.D{{"non-existent", bson.D{{"$exists", true}}}}},
				{"update", bson.D{{"$unset", bson.D{{"v", ""}}}}},
			},
		},
		"NonExistentExistsF": {
			command: bson.D{
				{"query", bson.D{{"non-existent", bson.D{{"$exists", false}}}}},
				{"update", bson.D{{"$unset", bson.D{{"v", ""}}}}},
			},
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/321",
			failsProviders:   []shareddata.Provider{shareddata.Unsets},
		},
		"ExistsTrue": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", true}}}}},
				{"update", bson.D{{"$unset", bson.D{{"v", ""}}}}},
			},
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/321",
			failsProviders:   []shareddata.Provider{shareddata.Unsets},
		},
		"ExistsFalse": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", false}}}}},
				{"update", bson.D{{"$unset", bson.D{{"v", ""}}}}},
			},
		},
		"UnsetNonExistentField": {
			command: bson.D{
				{"query", bson.D{{"_id", "double"}}},
				{"update", bson.D{{"$unset", bson.D{{"non-existent-field", ""}}}}},
			},
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/321",
			failsProviders:   []shareddata.Provider{shareddata.Scalars, shareddata.Doubles, shareddata.SmallDoubles},
		},
	}

	testFindAndModifyCompat(t, testCases)
}

func TestFindAndModifyCompatUpdateCurrentDate(t *testing.T) {
	t.Parallel()

	testCases := map[string]findAndModifyCompatTestCase{
		"NotDocument": {
			command: bson.D{
				{"query", bson.D{{"_id", "datetime"}}},
				{"update", bson.D{{"$currentDate", 1}}},
			},
			resultType: integration.EmptyResult,
		},
		"UnknownOption": {
			command: bson.D{
				{"query", bson.D{{"_id", "datetime"}}},
				{"update", bson.D{{"$currentDate", bson.D{{"v", bson.D{{"foo", int32(1)}}}}}}},
			},
			resultType:       integration.EmptyResult,
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/323",
			failsProviders: []shareddata.Provider{
				shareddata.Unsets, shareddata.Nulls, shareddata.DocumentsDocuments, shareddata.Mixed,
				shareddata.Regexes, shareddata.ArrayDocuments, shareddata.ArrayDoubles, shareddata.Int64s,
				shareddata.ObjectIDKeys, shareddata.DocumentsDoubles, shareddata.Strings, shareddata.Bools,
				shareddata.PostgresEdgeCases, shareddata.OverflowVergeDoubles, shareddata.ObjectIDs,
				shareddata.Composites, shareddata.DocumentsStrings, shareddata.DocumentsDeeplyNested, shareddata.Decimal128s,
				shareddata.Doubles, shareddata.Int32s, shareddata.Timestamps, shareddata.ArrayInt32s,
				shareddata.ArrayStrings, shareddata.ArrayRegexes, shareddata.ArrayAndDocuments,
				shareddata.SmallDoubles, shareddata.Binaries,
			},
		},
		"InvalidType": {
			command: bson.D{
				{"query", bson.D{{"_id", "datetime"}}},
				{"update", bson.D{{"$currentDate", bson.D{{"v", bson.D{{"$type", int32(1)}}}}}}},
			},
			resultType:       integration.EmptyResult,
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/323",
			failsProviders: []shareddata.Provider{
				shareddata.Unsets, shareddata.Nulls, shareddata.DocumentsDocuments, shareddata.Mixed,
				shareddata.Regexes, shareddata.ArrayDocuments, shareddata.ArrayDoubles, shareddata.Int64s,
				shareddata.ObjectIDKeys, shareddata.DocumentsDoubles, shareddata.Strings, shareddata.Bools,
				shareddata.PostgresEdgeCases, shareddata.OverflowVergeDoubles, shareddata.ObjectIDs,
				shareddata.Composites, shareddata.DocumentsStrings, shareddata.DocumentsDeeplyNested, shareddata.Decimal128s,
				shareddata.Doubles, shareddata.Int32s, shareddata.Timestamps, shareddata.ArrayInt32s,
				shareddata.ArrayStrings, shareddata.ArrayRegexes, shareddata.ArrayAndDocuments,
				shareddata.SmallDoubles, shareddata.Binaries,
			},
		},
		"UnknownType": {
			command: bson.D{
				{"query", bson.D{{"_id", "datetime"}}},
				{"update", bson.D{{"$currentDate", bson.D{{"v", bson.D{{"$type", "unknown"}}}}}}},
			},
			resultType:       integration.EmptyResult,
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/323",
			failsProviders: []shareddata.Provider{
				shareddata.Unsets, shareddata.Nulls, shareddata.DocumentsDocuments, shareddata.Mixed,
				shareddata.Regexes, shareddata.ArrayDocuments, shareddata.ArrayDoubles, shareddata.Int64s,
				shareddata.ObjectIDKeys, shareddata.DocumentsDoubles, shareddata.Strings, shareddata.Bools,
				shareddata.PostgresEdgeCases, shareddata.OverflowVergeDoubles, shareddata.ObjectIDs,
				shareddata.Composites, shareddata.DocumentsStrings, shareddata.DocumentsDeeplyNested, shareddata.Decimal128s,
				shareddata.Doubles, shareddata.Int32s, shareddata.Timestamps, shareddata.ArrayInt32s,
				shareddata.ArrayStrings, shareddata.ArrayRegexes, shareddata.ArrayAndDocuments,
				shareddata.SmallDoubles, shareddata.Binaries,
			},
		},
		"InvalidValue": {
			command: bson.D{
				{"query", bson.D{{"_id", "datetime"}}},
				{"update", bson.D{{"$currentDate", bson.D{{"v", 1}}}}},
			},
			resultType:       integration.EmptyResult,
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/323",
			failsProviders: []shareddata.Provider{
				shareddata.Unsets, shareddata.Nulls, shareddata.DocumentsDocuments, shareddata.Mixed,
				shareddata.Regexes, shareddata.ArrayDocuments, shareddata.ArrayDoubles, shareddata.Int64s,
				shareddata.ObjectIDKeys, shareddata.DocumentsDoubles, shareddata.Strings, shareddata.Bools,
				shareddata.PostgresEdgeCases, shareddata.OverflowVergeDoubles, shareddata.ObjectIDs,
				shareddata.Composites, shareddata.DocumentsStrings, shareddata.DocumentsDeeplyNested, shareddata.Decimal128s,
				shareddata.Doubles, shareddata.Int32s, shareddata.Timestamps, shareddata.ArrayInt32s,
				shareddata.ArrayStrings, shareddata.ArrayRegexes, shareddata.ArrayAndDocuments,
				shareddata.SmallDoubles, shareddata.Binaries,
			},
		},
	}

	testFindAndModifyCompat(t, testCases)
}

func TestFindAndModifyCompatUpdateRename(t *testing.T) {
	t.Parallel()

	testCases := map[string]findAndModifyCompatTestCase{
		"NotDocument": {
			command: bson.D{
				{"query", bson.D{{"_id", "int64"}}},
				{"update", bson.D{{"$rename", 1}}},
			},
			resultType: integration.EmptyResult,
		},
		"NonStringTargetField": {
			command: bson.D{
				{"query", bson.D{{"_id", "int64"}}},
				{"update", bson.D{{"$rename", bson.D{{"v", 0}}}}},
			},
			resultType: integration.EmptyResult,
		},
		"SameTargetField": {
			command: bson.D{
				{"query", bson.D{{"_id", "int64"}}},
				{"update", bson.D{{"$rename", bson.D{{"v", "v"}}}}},
			},
			resultType:       integration.EmptyResult,
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/324",
		},
		"DuplicateSource": {
			command: bson.D{
				{"query", bson.D{{"_id", "int64"}}},
				{"update", bson.D{{"$rename", bson.D{{"v", "w"}, {"v", "x"}}}}},
			},
			resultType:       integration.EmptyResult,
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/324",
		},
		"DuplicateTarget": {
			command: bson.D{
				{"query", bson.D{{"_id", "int64"}}},
				{"update", bson.D{{"$rename", bson.D{{"v", "w"}, {"x", "w"}}}}},
			},
			resultType: integration.EmptyResult,
		},
	}

	testFindAndModifyCompat(t, testCases)
}

// TestFindAndModifyCompatSort tests how various sort orders are handled.
func TestFindAndModifyCompatSort(t *testing.T) {
	t.Parallel()

	testCases := map[string]findAndModifyCompatTestCase{
		"DotNotation": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$in", bson.A{"array-documents-nested", "array-documents-nested-duplicate"}}}}}},
				{"update", bson.D{{"$set", bson.D{{"v.0.foo.0.bar", "baz"}}}}},
				{"sort", bson.D{{"v.0.foo", 1}, {"_id", 1}}},
			},
		},
		"DotNotationIndex": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$in", bson.A{"array-documents-nested", "array-documents-nested-duplicate"}}}}}},
				{"update", bson.D{{"$set", bson.D{{"v.0.foo.0.bar", "baz"}}}}},
				{"sort", bson.D{{"v.0.foo.0.bar", 1}, {"_id", 1}}},
			},
		},
		"DotNotationNonExistent": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$in", bson.A{"array-documents-nested", "array-documents-nested-duplicate"}}}}}},
				{"update", bson.D{{"$set", bson.D{{"v.0.foo.0.bar", "baz"}}}}},
				{"sort", bson.D{{"invalid.foo", 1}, {"_id", 1}}},
			},
		},
		"DotNotationMissingField": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$in", bson.A{"array-documents-nested", "array-documents-nested-duplicate"}}}}}},
				{"update", bson.D{{"$set", bson.D{{"v.0.foo.0.bar", "baz"}}}}},
				{"sort", bson.D{{"v..foo", 1}, {"_id", 1}}},
			},
			resultType:       integration.EmptyResult,
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/321",
		},
		"DollarPrefixedFieldName": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$in", bson.A{"array-documents-nested", "array-documents-nested-duplicate"}}}}}},
				{"update", bson.D{{"$set", bson.D{{"v.0.foo.0.bar", "baz"}}}}},
				{"sort", bson.D{{"$v.foo", 1}, {"_id", 1}}},
			},
			resultType:       integration.EmptyResult,
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/321",
		},
	}

	testFindAndModifyCompat(t, testCases)
}

func TestFindAndModifyCompatUpsert(t *testing.T) {
	t.Parallel()

	testCases := map[string]findAndModifyCompatTestCase{
		"UpsertNoSuchDocument": {
			command: bson.D{
				{"query", bson.D{{"_id", "no-such-doc"}}},
				{"update", bson.D{{"$set", bson.D{{"v", 43.13}}}}},
				{"upsert", true},
				{"new", true},
			},
		},
		"UpsertNoReplaceDocument": {
			command: bson.D{
				{"query", bson.D{{"_id", "no-such-doc"}}},
				{"update", bson.D{{"v", 43.13}}},
				{"upsert", true},
				{"new", true},
			},
		},
		"UpsertReplace": {
			command: bson.D{
				{"query", bson.D{{"_id", "double"}}},
				{"update", bson.D{{"v", 43.13}}},
				{"upsert", true},
			},
		},
		"UpsertReplaceReturnNew": {
			command: bson.D{
				{"query", bson.D{{"_id", "double"}}},
				{"update", bson.D{{"v", 43.13}}},
				{"upsert", true},
				{"new", true},
			},
		},
		"ExistsNew": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", false}}}}},
				{"upsert", true},
				{"update", bson.D{{"_id", "replaced"}, {"v", "replaced"}}},
				{"new", true},
			},
		},
		"ExistsFalse": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", false}}}}},
				{"upsert", true},
				{"update", bson.D{{"_id", "replaced"}, {"v", "replaced"}}},
			},
		},
		"UpdateID": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", true}}}}},
				{"upsert", true},
				{"update", bson.D{{"_id", "int32"}, {"v", "replaced"}}},
			},
			failsProviders: []shareddata.Provider{shareddata.Int32s},
		},
		"UpdateDifferentID": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", true}}}}},
				{"upsert", true},
				{"update", bson.D{{"_id", "replaced"}, {"v", "replaced"}}},
			},
			resultType: integration.EmptyResult, // _id must be an immutable field
		},
		"ExistsTrue": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", true}}}}},
				{"upsert", true},
				{"update", bson.D{{"v", "replaced"}}},
			},
		},
	}

	testFindAndModifyCompat(t, testCases)
}

func TestFindAndModifyCompatUpsertSet(t *testing.T) {
	t.Parallel()

	testCases := map[string]findAndModifyCompatTestCase{
		"Upsert": {
			command: bson.D{
				{"query", bson.D{{"_id", "double"}}},
				{"update", bson.D{{"$set", bson.D{{"v", 43.13}}}}},
				{"upsert", true},
			},
		},
		"UpsertNew": {
			command: bson.D{
				{"query", bson.D{{"_id", "double"}}},
				{"update", bson.D{{"$set", bson.D{{"v", 43.13}}}}},
				{"upsert", true},
				{"new", true},
			},
		},
		"UpsertNonExistent": {
			command: bson.D{
				{"query", bson.D{{"_id", "non-existent"}}},
				{"upsert", true},
				{"update", bson.D{{"$set", bson.D{{"v", "43"}}}}},
			},
		},
		"UpsertNewNonExistent": {
			command: bson.D{
				{"query", bson.D{{"_id", "non-existent"}}},
				{"upsert", true},
				{"update", bson.D{{"$set", bson.D{{"v", "43"}}}}},
				{"new", true},
			},
		},
		"NonExistentExistsFalse": {
			command: bson.D{
				{"query", bson.D{{"non-existent", bson.D{{"$exists", false}}}}},
				{"upsert", true},
				{"update", bson.D{{"$set", bson.D{{"v", "foo"}}}}},
			},
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/321",
			failsProviders:   []shareddata.Provider{shareddata.Strings},
		},
		"ExistsTrue": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", true}}}}},
				{"upsert", true},
				{"update", bson.D{{"$set", bson.D{{"v", "foo"}}}}},
			},
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/321",
			failsProviders:   []shareddata.Provider{shareddata.Strings},
		},
		"UpsertID": {
			command: bson.D{
				{"query", bson.D{{"_id", "non-existent"}}},
				{"upsert", true},
				{"update", bson.D{{"$set", bson.D{{"_id", "double"}}}}},
			},
			resultType: integration.EmptyResult, // _id must be an immutable field
		},
		"UpsertIDNoQuery": {
			command: bson.D{
				{"upsert", true},
				{"update", bson.D{{"$set", bson.D{{"_id", "int32"}, {"v", int32(2)}}}}},
			},
			failsProviders: []shareddata.Provider{shareddata.Int32s},
		},
		"UpsertExistingID": {
			command: bson.D{
				{"query", bson.D{{"_id", "int32"}}},
				{"upsert", true},
				{"update", bson.D{{"$set", bson.D{{"_id", "int32-1"}, {"v", int32(2)}}}}},
			},
			resultType: integration.EmptyResult,
		},
		"UpsertSameID": {
			command: bson.D{
				{"query", bson.D{{"_id", "int32"}}},
				{"upsert", true},
				{"update", bson.D{{"$set", bson.D{{"_id", "int32"}, {"v", int32(2)}}}}},
			},
		},
		"UpsertQueryOperatorEq": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$eq", "non-existent"}}}}},
				{"upsert", true},
				{"update", bson.D{{"$set", bson.D{{"new", "val"}}}}},
			},
		},
		"UpsertQueryOperatorMixed": {
			command: bson.D{
				{"query", bson.D{
					{"_id", bson.D{{"$eq", "non-existent"}}},
					{"v", bson.D{{"$lt", 43}}},
					{"non_existent", int32(0)},
				}},
				{"upsert", true},
				{"update", bson.D{{"$set", bson.D{{"new", "val"}}}}},
			},
		},
		"UpsertQueryObject": {
			command: bson.D{
				{"query", bson.D{{"_id", "non-existent"}, {"v", bson.D{{"k1", "v1"}}}}},
				{"upsert", true},
				{"update", bson.D{{"$set", bson.D{{"new", "val"}}}}},
			},
		},
		"UpsertQueryObjectNested": {
			command: bson.D{
				{"query", bson.D{{"_id", "non-existent"}, {"v", bson.D{{"k1", "v1"}, {"k2", bson.D{{"k21", "v21"}}}}}}},
				{"upsert", true},
				{"update", bson.D{{"$set", bson.D{{"new", "val"}}}}},
			},
		},
	}

	testFindAndModifyCompat(t, testCases)
}

func TestFindAndModifyCompatSetOnInsert(t *testing.T) {
	t.Parallel()

	testCases := map[string]findAndModifyCompatTestCase{
		"IDExists": {
			command: bson.D{
				{"query", bson.D{{"_id", "int32"}}},
				{"upsert", true},
				{"new", true},
				{"update", bson.D{{"$setOnInsert", bson.D{{"new", "val"}}}}},
			},
			providers:        []shareddata.Provider{shareddata.Int32s},
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/321",
			failsProviders:   []shareddata.Provider{shareddata.Int32s},
		},
		"IDNotExists": {
			command: bson.D{
				{"query", bson.D{{"_id", "non-existent"}}},
				{"upsert", true},
				{"new", true},
				{"update", bson.D{{"$setOnInsert", bson.D{{"new", "val"}}}}},
			},
		},
		"UpsertFalse": {
			command: bson.D{
				{"query", bson.D{{"_id", "non-existent"}}},
				{"upsert", false},
				{"new", true},
				{"update", bson.D{{"$setOnInsert", bson.D{{"new", "val"}}}}},
			},
		},
		"SetWithSetOnInsert": {
			command: bson.D{
				{"query", bson.D{{"_id", "non-existent"}}},
				{"upsert", true},
				{"new", true},
				{"update", bson.D{
					{"$set", bson.D{{"new", "val"}}},
					{"$setOnInsert", bson.D{{"v", int32(42)}}},
				}},
			},
		},
		"ApplySetSkipSetOnInsert": {
			command: bson.D{
				{"query", bson.D{{"_id", "int32"}}},
				{"upsert", true},
				{"new", true},
				{"update", bson.D{
					{"$set", bson.D{{"new", "val"}}},
					{"$setOnInsert", bson.D{{"v", int32(43)}}},
				}},
			},
			providers: []shareddata.Provider{shareddata.Int32s},
		},
	}

	testFindAndModifyCompat(t, testCases)
}

func TestFindAndModifyCompatUpsertUnset(t *testing.T) {
	t.Parallel()

	testCases := map[string]findAndModifyCompatTestCase{
		"NonExistentExistsT": {
			command: bson.D{
				{"query", bson.D{{"non-existent", bson.D{{"$exists", true}}}}},
				{"upsert", true},
				{"update", bson.D{
					{"$unset", bson.D{{"v", ""}}},
					{"$set", bson.D{{"_id", "upserted"}}}, // to have the same _id for target and compat
				}},
			},
		},
		"NonExistentExistsF": {
			command: bson.D{
				{"query", bson.D{{"non-existent", bson.D{{"$exists", false}}}}},
				{"upsert", true},
				{"update", bson.D{{"$unset", bson.D{{"v", ""}}}}},
			},
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/321",
			failsProviders:   []shareddata.Provider{shareddata.Unsets},
		},
		"ExistsTrue": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", true}}}}},
				{"upsert", true},
				{"update", bson.D{{"$unset", bson.D{{"v", ""}}}}},
			},
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/321",
			failsProviders:   []shareddata.Provider{shareddata.Unsets},
		},
		"ExistsFalse": {
			command: bson.D{
				{"query", bson.D{{"_id", bson.D{{"$exists", false}}}}},
				{"upsert", true},
				{"update", bson.D{
					{"$unset", bson.D{{"v", ""}}},
					{"$set", bson.D{{"_id", "upserted"}}}, // to have the same _id for target and compat
				}},
			},
		},
		"UnsetNonExistentField": {
			command: bson.D{
				{"query", bson.D{{"_id", "double"}}},
				{"upsert", true},
				{"update", bson.D{{"$unset", bson.D{{"non-existent-field", ""}}}}},
			},
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/321",
			failsProviders:   []shareddata.Provider{shareddata.Scalars, shareddata.Doubles, shareddata.SmallDoubles},
		},
	}

	testFindAndModifyCompat(t, testCases)
}

func TestFindAndModifyCompatRemove(t *testing.T) {
	t.Parallel()

	testCases := map[string]findAndModifyCompatTestCase{
		"Remove": {
			command: bson.D{
				{"query", bson.D{{"_id", "double"}}},
				{"remove", true},
			},
		},
		"RemoveEmptyQueryResult": {
			command: bson.D{
				{
					"query",
					bson.D{{
						"$and",
						bson.A{
							bson.D{{"v", bson.D{{"$gt", 0}}}},
							bson.D{{"v", bson.D{{"$lt", 0}}}},
						},
					}},
				},
				{"remove", true},
			},
		},
	}

	testFindAndModifyCompat(t, testCases)
}

func TestFindAndModifyCompatReplacementDoc(t *testing.T) {
	t.Parallel()

	testCases := map[string]findAndModifyCompatTestCase{
		"Basic": {
			command: bson.D{
				{"update", bson.D{{"v", int32(43)}}},
			},
		},
		"EmptyDoc": {
			command: bson.D{
				{"update", bson.D{}},
			},
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/321",
			failsProviders:   []shareddata.Provider{shareddata.Unsets},
		},
		"FilterAndUpsertTrue": {
			command: bson.D{
				{"query", bson.D{{"_id", "non-existent"}}},
				{"update", bson.D{{"v", int32(43)}}},
				{"upsert", true},
			},
		},
		"WithUpdateOp": {
			command: bson.D{
				{"update", bson.D{{"v", int32(43)}, {"$set", bson.D{{"test", int32(0)}}}}},
			},
			resultType: integration.EmptyResult,
		},
		"SameId": {
			command: bson.D{
				{"query", bson.D{{"_id", "int32"}}},
				{"update", bson.D{{"_id", "int32"}, {"v", int32(43)}}},
			},
		},
		"DifferentId": {
			command: bson.D{
				{"query", bson.D{{"_id", "int32"}}},
				{"update", bson.D{{"_id", "non-existent"}, {"v", int32(43)}}},
			},
		},
	}

	testFindAndModifyCompat(t, testCases)
}

// findAndModifyCompatTestCase describes findAndModify compatibility test case.
type findAndModifyCompatTestCase struct {
	command    bson.D
	resultType integration.CompatTestCaseResultType // defaults to NonEmptyResult
	providers  []shareddata.Provider                // defaults to shareddata.AllProviders()

	failsForFerretDB string
	failsProviders   []shareddata.Provider // use only if failsForFerretDB is set, defaults to all providers
}

// testFindAndModifyCompat tests findAndModify compatibility test cases.
func testFindAndModifyCompat(t *testing.T, testCases map[string]findAndModifyCompatTestCase) {
	t.Helper()

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Helper()

			t.Parallel()

			providers := shareddata.AllProviders()
			if tc.providers != nil {
				providers = tc.providers
			}

			s := setup.SetupCompatWithOpts(t, &setup.SetupCompatOpts{Providers: providers})

			ctx, targetCollections, compatCollections := s.Ctx, s.TargetCollections, s.CompatCollections

			failsProviders := make([]string, len(tc.failsProviders))
			for i, p := range tc.failsProviders {
				failsProviders[i] = p.Name()
			}

			var nonEmptyResults bool
			for i := range targetCollections {
				targetCollection := targetCollections[i]
				compatCollection := compatCollections[i]

				t.Run(targetCollection.Name(), func(tt *testing.T) {
					var t testing.TB = tt

					// a workaround to get provider name by using the part after last `_`,
					// e.g. `ArrayAndDocuments` from `TestFindAndModifyCompatDotNotation-NoIndex_ArrayAndDocuments`
					str := strings.Split(targetCollection.Name(), "_")
					providerName := str[len(str)-1]

					failsForCollection := len(tc.failsProviders) == 0 || slices.Contains(failsProviders, providerName)

					if tc.failsForFerretDB != "" && failsForCollection {
						t = setup.FailsForFerretDB(tt, tc.failsForFerretDB)
					}

					t.Helper()

					targetCommand := bson.D{{"findAndModify", targetCollection.Name()}}
					targetCommand = append(targetCommand, tc.command...)
					if targetCommand.Map()["sort"] == nil {
						targetCommand = append(targetCommand, bson.D{{"sort", bson.D{{"_id", 1}}}}...)
					}

					compatCommand := bson.D{{"findAndModify", compatCollection.Name()}}
					compatCommand = append(compatCommand, tc.command...)
					if compatCommand.Map()["sort"] == nil {
						compatCommand = append(compatCommand, bson.D{{"sort", bson.D{{"_id", 1}}}}...)
					}

					var targetMod, compatMod bson.D
					var targetErr, compatErr error
					targetErr = targetCollection.Database().RunCommand(ctx, targetCommand).Decode(&targetMod)
					compatErr = compatCollection.Database().RunCommand(ctx, compatCommand).Decode(&compatMod)

					if targetErr != nil {
						t.Logf("Target error: %v", targetErr)
						t.Logf("Compat error: %v", compatErr)

						// error messages are intentionally not compared
						integration.AssertMatchesError(t, compatErr, targetErr)

						return
					}
					require.NoError(t, compatErr, "compat error; target returned no error")

					integration.AssertEqualDocuments(t, compatMod, targetMod)

					// To make sure that the results of modification are equal,
					// find all the documents in target and compat collections and compare that they are the same
					opts := options.Find().SetSort(bson.D{{"_id", 1}})
					targetCursor, targetErr := targetCollection.Find(ctx, bson.D{}, opts)
					compatCursor, compatErr := compatCollection.Find(ctx, bson.D{}, opts)

					if targetCursor != nil {
						defer targetCursor.Close(ctx)
					}
					if compatCursor != nil {
						defer compatCursor.Close(ctx)
					}

					if targetErr != nil {
						t.Logf("Target error: %v", targetErr)
						targetErr = integration.UnsetRaw(t, targetErr)
						compatErr = integration.UnsetRaw(t, compatErr)
						assert.Equal(t, compatErr, targetErr)
						return
					}
					require.NoError(t, compatErr, "compat error; target returned no error")

					targetRes := integration.FetchAll(t, ctx, targetCursor)
					compatRes := integration.FetchAll(t, ctx, compatCursor)

					t.Logf("Compat (expected) IDs: %v", integration.CollectIDs(t, compatRes))
					t.Logf("Target (actual)   IDs: %v", integration.CollectIDs(t, targetRes))
					integration.AssertEqualDocumentsSlice(t, compatRes, targetRes)

					if len(targetRes) > 0 || len(compatRes) > 0 {
						nonEmptyResults = true
					}
				})
			}

			switch tc.resultType {
			case integration.NonEmptyResult:
				if tc.failsForFerretDB != "" {
					return
				}

				assert.True(t, nonEmptyResults, "expected non-empty results")
			case integration.EmptyResult:
				assert.False(t, nonEmptyResults, "expected empty results")
			default:
				t.Fatalf("unknown result type %v", tc.resultType)
			}
		})
	}
}
