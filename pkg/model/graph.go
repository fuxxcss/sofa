package model

import (
	"log"
)

// graph = level 0
type Graph struct {

	prev []*Vertex
	next []*Vertex
}

type Vertex struct {

	data *Token
	prev  *Vertex
	next  []*Vertex
}

func NewGraph() *Graph {

	graph := new(Graph)
	graph.prev = make([]*Vertex, 0)
	graph.next = make([]*Vertex, 0)

	return graph
}

func (self *Graph) AddVertex(data *Token) *Vertex{

	vertex := new(Vertex)
	vertex.data = data
	vertex.next = make([]*Vertex, 0)

	return vertex
}

// public
func (self *Graph) Match(graph *Graph) bool {

	// token level num
	matchNum := make(map[TokenLevel]int, 0)
	hasNum := make(map[TokenLevel]int, 0)

	// token level slice
	matchSlice := make(map[TokenLevel][]*Token, 0)
	hasSlice := make(map[TokenLevel][]*Token, 0)

	// to match
	for _, prev := range self.prev {

		// update match
		level := prev.data.Level
		_, ok := matchNum[level]

		if ok {
			matchNum[level] ++
			matchSlice[level] = append(matchSlice[level], prev.data)
		}else {
			matchNum[level] = 1
			matchSlice[level] = make([]*Token, 0)
		}

	}

	// graph has
	for _, next := range graph.next {

		// update has
		level := next.data.Level
		_, ok := hasNum[level]

		if ok {
			hasNum[level] ++
			hasSlice[level] = append(hasSlice[level], next.data)
		}else {
			hasNum[level] = 1
			hasSlice[level] = make([]*Token, 0)
		}

	}

	// cannot match
	for level, num := range matchNum {

		if hasNum[level] < num {
			return false
		}
	}

	// match self -> graph
	for level, slice := range matchSlice {

		// match tokens
		for index, token := range slice {
			token.Text = hasSlice[level][index].Text
		}
	}

	return true
}

// debug
func (self *Graph) Debug() {

	log.Println("==== graph prev ====")

	for _, prev := range self.prev {

		str := prev.data.Text
		size := len(str)

		if len(prev.next) == 0 {
			log.Printf("%s(size=%d)\n", str, size)
			continue
		}

		for _, next := range prev.next {
			log.Printf("%s(size=%d) ---> %s(size=%d)\n", str, size, next.data.Text, len(next.data.Text))
		}
	}

	log.Println("==== graph next ====")

	for _, next := range self.next {

		str := next.data.Text
		size := len(str)

		if len(next.next) == 0 {
			log.Printf("%s(size=%d)\n", str, size)
			continue
		}

		for _, next := range next.next {
			log.Printf("%s(size=%d) ---> %s(size=%d)\n", str, size, next.data.Text, len(next.data.Text))
		}
	}
}





