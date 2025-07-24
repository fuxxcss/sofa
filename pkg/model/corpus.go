package model

import (
	"crypto/md5"
	"log"
	"strings"

	"github.com/fuxxcss/redi2fuzz/pkg/utils"
)

/*
 * Definition
 */

// corpus json
const (
	CorpusPath string = "queue/corpus-json"
)

// corpus len
const (
	CORPUS_MINLEN int = 15
	CORPUS_MAXLEN int = 45
)

type Corpus struct {
	weight int64
	order   []*Line
}

/*
 * Function
 */

// public
func NewCorpus() *Corpus {

	corpus := new(Corpus)
	corpus.weight = 0
	corpus.order = make([]*Line, 0)

	return corpus
}

// public
func (self *Corpus) AddFile(file string) []*Line {

	ret := make([]*Line, 0)

	// split line
	lines := strings.Split(file, LineSep)

	for _, line := range lines {

		// md5
		sum := md5.Sum([]byte(line))
		hash := string(sum[:])

		// new line
		new := NewLine(line, hash)

		self.order = append(self.order, new)
		ret = append(ret, new)
	}

	return ret
}

func (self *Corpus) Mutate() []*Line {

	// mutated len
	length := utils.RandInt(CORPUS_MAXLEN-CORPUS_MINLEN) + CORPUS_MINLEN

	ret := make([]*Line, 0)

	for i := 0; i < length; {

		// select one line
		line := self.Select()

		if line == nil {
			continue
		}

		// repair line
		isRepaired := line.Repair(ret)

		if !isRepaired {
			continue
		}

		// mutate line
		line.Mutate()
		ret = append(ret, line)

		// one line is ready
		i ++
	}

	return ret
}

// public
func (self *Corpus) Select() *Line {

	// init corpus weight
	if self.weight == 0 {
		for _, line := range self.order {
			self.weight += line.Weight
		}
	}

	// roulette wheel selection
	rand := utils.RandFloat() * float64(self.weight)
	var sum int64 = 0

	// select line
	for _, line := range self.order {

		sum += line.Weight

		if float64(sum) > rand {
			return line
		}
	}

	return nil
}

// debug
func (self *Corpus) Debug() {

	log.Printf("Corpus Num: %d\n", len(self.order))

	for _, line := range self.order {
		line.Debug()
	}
}
