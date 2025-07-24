package analyze

import (
	"fmt"
	"log"
	"os"

	"github.com/fuxxcss/redi2fuzz/pkg/db"
	"github.com/fuxxcss/redi2fuzz/pkg/utils"
)

func Analyze(target utils.TargetType, path string) {

	// Analyze Target (redis, keydb, redis-stack)
	feature := utils.Targets[target]
	context, err := os.ReadFile(path)

	if err != nil {
		log.Fatalln("err: bug file failed.")
	}

	// interface
	var DBtarget db.DB

	switch target {
	// Redi
	case utils.REDI_REDIS, utils.REDI_KEYDB, utils.REDI_STACK:
		DBtarget = db.NewRedi(feature)
	}

	// StartUp target first
	err = DBtarget.StartUp()
	defer DBtarget.ShutDown()

	if err != nil {
		log.Println("err: db startup failed.")
		return
	}

	// from json
	cj := make(utils.CrashJson, 0)
	err = cj.FromJson(context)

	if err != nil {
		log.Fatalln("err: json decode failed.")
	}

	// test bug
	index := -1
	var bug []string

	for i, line := range cj {

		// execute each line
		DBtarget.Execute(line)

		alive := DBtarget.CheckAlive()

		// crash
		if !alive {

			index = i
			bug = line
			break
		}
	}

	// trigger bug
	if index >= 0 {

		fmt.Printf("line %d trigger bug\n", index+1)
		fmt.Println(bug)
		fmt.Println(DBtarget.Stderr())

	} else {
		fmt.Println("not a bug")
	}

}
