//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package scorers

import (
	"fmt"

	"github.com/blevesearch/bleve/search"
)

type ConstantScorer struct {
	constant               float64
	boost                  float64
	explain                bool
	queryNorm              float64
	queryWeight            float64
	queryWeightExplanation *search.Explanation
}

func NewConstantScorer(constant float64, boost float64, explain bool) *ConstantScorer {
	rv := ConstantScorer{
		explain:     explain,
		queryWeight: 1.0,
		constant:    constant,
		boost:       boost,
	}

	return &rv
}

func (s *ConstantScorer) Weight() float64 {
	sum := s.boost
	return sum * sum
}

func (s *ConstantScorer) SetQueryNorm(qnorm float64) {
	s.queryNorm = qnorm

	// update the query weight
	s.queryWeight = s.boost * s.queryNorm

	if s.explain {
		childrenExplanations := make([]*search.Explanation, 2)
		childrenExplanations[0] = &search.Explanation{
			Value:   s.boost,
			Message: "boost",
		}
		childrenExplanations[1] = &search.Explanation{
			Value:   s.queryNorm,
			Message: "queryNorm",
		}
		s.queryWeightExplanation = &search.Explanation{
			Value:    s.queryWeight,
			Message:  fmt.Sprintf("ConstantScore()^%f, product of:", s.boost),
			Children: childrenExplanations,
		}
	}
}

func (s *ConstantScorer) Score(id string) *search.DocumentMatch {
	var scoreExplanation *search.Explanation

	score := s.constant

	if s.explain {
		scoreExplanation = &search.Explanation{
			Value:   score,
			Message: fmt.Sprintf("ConstantScore()"),
		}
	}

	// if the query weight isn't 1, multiply
	if s.queryWeight != 1.0 {
		score = score * s.queryWeight
		if s.explain {
			childExplanations := make([]*search.Explanation, 2)
			childExplanations[0] = s.queryWeightExplanation
			childExplanations[1] = scoreExplanation
			scoreExplanation = &search.Explanation{
				Value:    score,
				Message:  fmt.Sprintf("weight(^%f), product of:", s.boost),
				Children: childExplanations,
			}
		}
	}

	rv := search.DocumentMatch{
		ID:    id,
		Score: score,
	}
	if s.explain {
		rv.Expl = scoreExplanation
	}

	return &rv
}
