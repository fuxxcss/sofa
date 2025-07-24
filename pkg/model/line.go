package model

import (
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/fuxxcss/redi2fuzz/pkg/utils"
)

// global
var LineSep string
var TokenSep string

// token level
type TokenLevel int

const (
	// value, str
	TOKEN_LEVEL_value TokenLevel = iota
	TOKEN_LEVEL_str
	// cmd, ...
	TOKEN_LEVEL_0
	// key, field ...
	TOKEN_LEVEL_1
	TOKEN_LEVEL_2
	TOKEN_LEVEL_3
)

// line score
const (
	LINE_SCORE_CREATE int64 = 2
	LINE_SCORE_DELETE int64 = 2
	LINE_SCORE_KEEP   int64 = 1
)

type Line struct {

	// public
	Weight int64

	// private
	graph  *Graph
	tokens []*Token
}

type Token struct {
	Level TokenLevel
	Text  string
}

// public
func NewLine(str, hash string) *Line {

	Line := new(Line)

	Line.graph = NewGraph()
	Line.tokens = make([]*Token, 0)
	Line.Weight = 0

	// split tokens
	tokens := strings.Split(str, TokenSep)

	for _, token := range tokens {

		// skip empty
		if token == "" {
			continue
		}

		// default 0
		tokenLevel := TOKEN_LEVEL_0

		// new token
		newToken := new(Token)

		// value, str
		_, intErr := strconv.Atoi(token)
		_, floatErr := strconv.ParseFloat(token, 64)
		isContain := strings.Contains(token, "\"")

		if intErr == nil || floatErr == nil {
			tokenLevel = TOKEN_LEVEL_value
		}

		if isContain {
			tokenLevel = TOKEN_LEVEL_str
		}

		newToken.Level = tokenLevel
		newToken.Text = token

		Line.tokens = append(Line.tokens, newToken)
	}

	return Line
}

// build model, graph
// update Weight, snapshot
func (self *Line) Build(new Snapshot) error {

	old := oldSnapshot

	// loop create, keep
	for from, toSlice := range new {

		var fromVertex *Vertex

		// create from
		if !old.Contains(from) {

			// from token
			fromToken := self.Contains(&from)

			if fromToken == nil {
				return errors.New("Create fromToken Nil.")
			}

			fromToken.Level = from.Level

			// from vertex
			fromVertex = self.graph.AddVertex(fromToken)

			// add score
			self.Weight += LINE_SCORE_CREATE

			// add vertex
			self.graph.next = append(self.graph.next, fromVertex)

			// keep from
		} else {

			// from token
			fromToken := self.Contains(&from)

			if fromToken == nil {
				continue
			}

			fromToken.Level = from.Level

			// from vertex
			fromVertex = self.graph.AddVertex(fromToken)

			// add score
			self.Weight += LINE_SCORE_KEEP

			// add vertex
			self.graph.prev = append(self.graph.prev, fromVertex)
		}

		for _, to := range toSlice {

			// create to
			if !old.Contains(to) {

				// to token
				toToken := self.Contains(&to)

				if toToken == nil {
					return errors.New("Create toToken Nil.")
				}

				toToken.Level = to.Level

				// to vertex
				toVertex := self.graph.AddVertex(toToken)
				toVertex.prev = fromVertex

				// add score
				self.Weight += LINE_SCORE_CREATE

				// add vertex
				self.graph.next = append(self.graph.next, toVertex)
				fromVertex.next = append(fromVertex.next, toVertex)

				// keep to
			} else {

				// remove to
				old[from] = old.Delete(from, to)

				// to token
				toToken := self.Contains(&to)

				if toToken == nil {
					continue
				}

				toToken.Level = to.Level

				// to vertex
				toVertex := self.graph.AddVertex(toToken)
				toVertex.prev = fromVertex

				// add score
				self.Weight += LINE_SCORE_KEEP

				// add vertex
				self.graph.prev = append(self.graph.prev, toVertex)
				fromVertex.next = append(fromVertex.next, toVertex)
			}
		}

		// remove from
		if len(old[from]) == 0 {
			delete(old, from)
		}
	}

	// only deleted
	for from, toSlice := range old {

		// from token
		fromToken := self.Contains(&from)

		if fromToken == nil {
			continue
		}

		fromToken.Level = from.Level

		// from vertex
		fromVertex := self.graph.AddVertex(fromToken)

		// add score
		self.Weight += LINE_SCORE_DELETE

		// add vertex
		self.graph.prev = append(self.graph.prev, fromVertex)

		for _, to := range toSlice {

			// to token
			toToken := self.Contains(&to)

			if toToken == nil {
				continue
			}

			toToken.Level = to.Level

			// to vertex
			toVertex := self.graph.AddVertex(toToken)
			toVertex.prev = fromVertex

			// add score
			self.Weight += LINE_SCORE_DELETE

			// add vertex
			self.graph.prev = append(self.graph.prev, toVertex)
			fromVertex.next = append(fromVertex.next, toVertex)
		}
	}

	// update snapshot
	oldSnapshot = new

	return nil
}

// public
func (self *Line) Repair(lines []*Line) bool {

	if len(self.graph.prev) == 0 {
		return true
	}

	// match graph
	for _, line := range lines {

		// match succeed
		if self.graph.Match(line.graph) {
			return true
		}
	}

	// match failed
	return false
}

// public
func (self *Line) Mutate() {

	// mutate str, value
	for _, token := range self.tokens {

		switch token.Level {

		// mutate str
		case TOKEN_LEVEL_str:
			token.Text = MutateStr(token.Text)
			
		// mutate value
		case TOKEN_LEVEL_value:
			item := utils.RandInt(len(utils.InterestingValue))
			token.Text = utils.InterestingValue[item]
		}
	}

	// mutate level 123...
	for _, next := range self.graph.next {

		text := next.data.Text
		next.data.Text = MutateStr(text)
	}

}

func MutateStr(str string) string {

	item := utils.RandInt(len(utils.InterestingStr))
	chosen := utils.InterestingStr[item]

	switch item {

	// empty
	case utils.InterestEmpty:
		return chosen

	// special
	case utils.InterestSpecial:
		special := utils.RandInt(len(chosen))
		return str + string(chosen[special])

	// null, terminal, hex, short str
	default:
		return str + chosen
	}

}

// public
func (self *Line) Text() []string {

	ret := make([]string, 0)

	for _, token := range self.tokens {
		ret = append(ret, token.Text)
	}

	return ret
}

// public
func (self *Line) Contains(token *Token) *Token {

	for _, t := range self.tokens {

		// skip value
		if t.Level == TOKEN_LEVEL_value || t.Level == TOKEN_LEVEL_str {
			continue
		}

		// contains
		if strings.Contains(t.Text, token.Text) {
			return t
		}
	}

	return nil
}

// debug
func (self *Line) Debug() {

	log.Println("==== Line Debug ====")

	var str string

	texts := self.Text()

	for _, text := range texts {
		str += text + TokenSep
	}

	log.Printf("line: %s\n", str)
	log.Printf("weight: %d\n", self.Weight)
	self.graph.Debug()

	log.Println("==== Line Debug ====")
}
