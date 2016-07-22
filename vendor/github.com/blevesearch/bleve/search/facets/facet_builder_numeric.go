//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package facets

import (
	"sort"

	"github.com/blevesearch/bleve/index"
	"github.com/blevesearch/bleve/numeric_util"
	"github.com/blevesearch/bleve/search"
)

type numericRange struct {
	min *float64
	max *float64
}

type NumericFacetBuilder struct {
	size       int
	field      string
	termsCount map[string]int
	total      int
	missing    int
	ranges     map[string]*numericRange
}

func NewNumericFacetBuilder(field string, size int) *NumericFacetBuilder {
	return &NumericFacetBuilder{
		size:       size,
		field:      field,
		termsCount: make(map[string]int),
		ranges:     make(map[string]*numericRange, 0),
	}
}

func (fb *NumericFacetBuilder) AddRange(name string, min, max *float64) {
	r := numericRange{
		min: min,
		max: max,
	}
	fb.ranges[name] = &r
}

func (fb *NumericFacetBuilder) Update(ft index.FieldTerms) {
	terms, ok := ft[fb.field]
	if ok {
		for _, term := range terms {
			// only consider the values which are shifted 0
			prefixCoded := numeric_util.PrefixCoded(term)
			shift, err := prefixCoded.Shift()
			if err == nil && shift == 0 {
				i64, err := prefixCoded.Int64()
				if err == nil {
					f64 := numeric_util.Int64ToFloat64(i64)

					// look at each of the ranges for a match
					for rangeName, r := range fb.ranges {

						if (r.min == nil || f64 >= *r.min) && (r.max == nil || f64 < *r.max) {

							existingCount, existed := fb.termsCount[rangeName]
							if existed {
								fb.termsCount[rangeName] = existingCount + 1
							} else {
								fb.termsCount[rangeName] = 1
							}
							fb.total++
						}
					}
				}
			}
		}
	} else {
		fb.missing++
	}
}

func (fb *NumericFacetBuilder) Result() *search.FacetResult {
	rv := search.FacetResult{
		Field:   fb.field,
		Total:   fb.total,
		Missing: fb.missing,
	}

	rv.NumericRanges = make([]*search.NumericRangeFacet, 0, len(fb.termsCount))

	for term, count := range fb.termsCount {
		numericRange := fb.ranges[term]
		tf := &search.NumericRangeFacet{
			Name:  term,
			Count: count,
			Min:   numericRange.min,
			Max:   numericRange.max,
		}

		rv.NumericRanges = append(rv.NumericRanges, tf)
	}

	sort.Sort(rv.NumericRanges)

	// we now have the list of the top N facets
	if fb.size < len(rv.NumericRanges) {
		rv.NumericRanges = rv.NumericRanges[:fb.size]
	}

	notOther := 0
	for _, nr := range rv.NumericRanges {
		notOther += nr.Count
	}
	rv.Other = fb.total - notOther

	return &rv
}
