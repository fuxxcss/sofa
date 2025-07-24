package fuzz

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/fuxxcss/redi2fuzz/pkg/db"
	"github.com/fuxxcss/redi2fuzz/pkg/model"
	"github.com/fuxxcss/redi2fuzz/pkg/utils"
)

// export
func Fuzz(target utils.TargetType) {

	// Fuzz Target (redis, keydb, redis-stack)
	feature := utils.Targets[target]
	queue := feature[utils.QUEUE_PATH]

	// interface
	var DBtarget db.DB

	switch target {
	// Redi
	case utils.REDI_REDIS, utils.REDI_KEYDB, utils.REDI_STACK:
		DBtarget = db.NewRedi(feature)
	}

	// StartUp target first
	err := DBtarget.StartUp()
	defer DBtarget.ShutDown()

	if err != nil {
		log.Println("err: db startup failed.")
		return
	}

	chanExit := make(chan struct{})

	// exit control
	go signalCtl(chanExit)

	// fuzz server
	go fuzzServer(DBtarget, queue)

	// exit
	<-chanExit

}

// static
func signalCtl(chanExit chan<- struct{}) {

	chanSig := make(chan os.Signal, 1)
	signal.Notify(chanSig, os.Interrupt, syscall.SIGTERM)

	<-chanSig

	fmt.Println("[*] Fuzz Proc is Killed.")

	chanExit <- struct{}{}

}

// private
func fuzzServer(target db.DB, queue string) {

	// init corpus
	corpus := model.NewCorpus()
	fmt.Println("[*] init corpus...")

	var lines []*model.Line

	filepath.Walk(queue, func(file string, info os.FileInfo, err error) error {

		if err != nil {
			log.Fatalln("err: queue path wrong.")
		}

		if info.IsDir() {
			return nil
		}

		// read file
		content, err := os.ReadFile(file)

		if err != nil {
			log.Println("err: read queue failed.", file)
		}

		lines = corpus.AddFile(string(content))

		// clean up database first
		err = target.CleanUp()

		if err != nil {
			log.Println("err: clean up failed")
		}

		// fuzz loop
		fuzzLoop(target, lines)

		return nil
	})

	corpus.Debug()

	log.Fatal()
	fmt.Println("[*] corpus ok")

	// mutate loop
	tryCnt := 0

	for {

		// mutated line
		mutated := corpus.Mutate()

		// clean up database
		err := target.CleanUp()

		if err != nil {
			log.Println("clean up failed")
		}

		for index, line := range mutated {

			// execute
			args := line.Text()
			state, err := target.Execute(args)

			// print
			fmt.Println(utils.Divide)
			fmt.Printf("fuzz count: %d\n", tryCnt)
			fmt.Printf("fuzz line: %s\n", args)

			if err != nil {
				fmt.Println(err)
			}

			// crash
			if state == utils.STATE_CRASH {
				fuzzCrash(target, mutated, index)
			}
		}

		tryCnt++
	}
}

// private
func fuzzLoop(target db.DB, lines []*model.Line) {

	// snapshot
	var snapshot model.Snapshot

	okCnt := len(lines)

	for index, line := range lines {

		// execute
		args := line.Text()
		state, err := target.Execute(args)

		switch state {

		// ok
		case utils.STATE_OK:

			// collect snapshot
			snapshot, err = target.Collect()

			if err != nil {
				log.Println("err: Collect Snapshot ", err)
			}

			// build line
			err = line.Build(snapshot)

			if err != nil {
				log.Println("err: Build Line ", err)
			}

		// err
		case utils.STATE_ERR:

			fmt.Println("[*] state error:", args)
			fmt.Println(err)
			okCnt--

		// crash
		case utils.STATE_CRASH:

			fuzzCrash(target, lines, index)
		}

	}

	// bad testcase
	if okCnt < model.CORPUS_MINLEN {

		fmt.Println("[*] bad queue.")
	}

}

// private
func fuzzCrash(target db.DB, lines []*model.Line, index int) {

	t := time.Now().UnixNano()
	name := "poc/" + strconv.FormatInt(t, 10)

	file, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)

	// don't miss crash
	if err != nil {
		crashPrint(lines, index)
	}

	// log crash in json
	size := len(lines)
	cj := make(utils.CrashJson, size)

	for i, line := range lines {
		cj[i] = line.Text()
	}

	// to json
	bytes, err := cj.ToJson()

	if err != nil {
		crashPrint(lines, index)
	}

	file.Write(bytes)

	// restart
	target.Restart()
}

func crashPrint(lines []*model.Line, index int) {

	// alert
	alert := "[*] Found a crash :)\n"
	indexStr := strconv.Itoa(index)
	alert += "index ==> " + indexStr + "\n"

	var crash string

	for _, line := range lines {

		var str string
		texts := line.Text()

		for _, text := range texts {
			str += text + model.TokenSep
		}

		crash += str + model.LineSep
	}

	fmt.Println(alert)
	log.Fatalln(crash)
}
