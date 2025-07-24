package model

import (
	"log"
	"slices"
)

// global
var oldSnapshot Snapshot

// Snapshot
type Snapshot map[Token][]Token

func NewSnapshot() Snapshot {

	snapshot := make(Snapshot, 0)

	return snapshot
}

func (self *Snapshot) Debug() {

	log.Println("==== snapshot ====")

	for k, slice := range *self {

		textK := k.Text
		sizeK := len(textK)

		for _, v := range slice {

			textV := v.Text
			sizeV := len(textV)

			log.Printf("%s(size=%d,level=%d) ---> %s(size=%d,level=%d)\n", textK, sizeK, k.Level, textV, sizeV, v.Level)
		}
	}
}

func (self *Snapshot) Contains(token Token) bool {

	_, ok := (*self)[token]

	// contain token
	if ok {
		return true
	}

	for _, tokens := range *self {

		// contain token
		if slices.Contains(tokens, token) {
			return true
		}
	}

	return false
}

func (self *Snapshot) Delete(k, v Token) []Token {

	slice := (*self)[k]

	// shift
	i := 0
	for _, t := range slice {
		if t != v {
			slice[i] = t
			i++
		}
	}

	return slice[:i]
}
