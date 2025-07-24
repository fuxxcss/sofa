package db

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/fuxxcss/redi2fuzz/pkg/utils"
	"github.com/fuxxcss/redi2fuzz/pkg/model"
	"github.com/redis/go-redis/v9"
)

/*
 * Redi Definition
 */

/*	Redi Struct	*/
type Redi struct {

	// proc ...
	path string
	args []string
	stderr bytes.Buffer
	proc *exec.Cmd

	// runtime ...
	client *redis.Client
	ctx    context.Context
}

const (
	RediLineSep	 string = "\n"
	RediTokenSep string = " "
)


/*
 * Redi Functions
 */

func NewRedi(feature utils.TargetFeature) *Redi {

	// set global sep
	model.LineSep = RediLineSep
	model.TokenSep = RediTokenSep

	redi := new(Redi)

	// path, port
	var path, port string

	path = feature[utils.TARGET_PATH]
	port = feature[utils.TARGET_PORT]

	// cannot find path
	_, err := os.Stat(path)

	if err != nil {
		log.Fatalf("err: %s %v", path, err)
	}

	redi.path = path
	
	// redi runtime
	redi.client = redis.NewClient(&redis.Options{
		Addr:     "localhost:" + port,
		Password: "",
		DB:       0,
	})

	redi.ctx = context.Background()

	// check alive
	alive := redi.CheckAlive()

	// already startup, shutdown first
	if alive {
		redi.client.Do(redi.ctx,"shutdown")
	}

	// redi args
	redi.args = []string{
		// port
		"--port" + " " + port,
	}

	return redi

}

/*
 * Redi Interface
 */

// Redi Struct
// public
func (self *Redi) StartUp() error {

	self.proc = exec.Command(self.path, self.args...)
	self.proc.Stderr = &self.stderr

	// error
	err := self.proc.Start()

	// startup failed
	if err != nil {
		return err
	}

	// waiting redi startup
	fmt.Println("[*] waiting redi startup...")
	for {
		alive := self.CheckAlive()
		if alive {
			break
		}
	}

	// succeed
	fmt.Printf("[*] Redi %s StartUp.\n", self.path)

	return nil
}

// public
func (self *Redi) Restart() error {

	err := self.proc.Start()

	// restart failed
	if err != nil {
		return err
	}

	// waiting redi restart
	fmt.Println("[*] waiting redi restart...")
	for {
		alive := self.CheckAlive()
		if alive {
			break
		}
	}

	// db succeed
	fmt.Printf("[*] Redi %v ReStart.\n", self.path)

	return nil
}

// public
func (self *Redi) ShutDown() {

	// kill redi
	self.proc.Process.Kill()

}

// public
func (self *Redi) CheckAlive() bool {

	// redi state
	_, err := self.client.Ping(self.ctx).Result()

	// redi is not alive
	if err != nil {
		return false
	}

	return true
}

// public
func (self *Redi) CleanUp() error {

	_, err := self.client.FlushAll(self.ctx).Result()

	// flushall failed
	if err != nil {
		return err
	}

	return nil
}

// public
func (self *Redi) Execute(tokens []string) (utils.TargetState, error) {

	// marshal string
	args := []interface{}{}

	for _, token := range tokens {
		args = append(args, token)
	}

	// state
	state := utils.STATE_OK

	_, err := self.client.Do(self.ctx, args...).Result()

	// execute failed
	if err != nil && err != redis.Nil {

		// execute error
		if self.CheckAlive() {
			state = utils.STATE_ERR

		// crash
		} else {
			state = utils.STATE_CRASH
		}
	}

	return state, err

}

// public
func (self *Redi) Collect() (model.Snapshot, error) {

	// snapshot
	snapshot := make(model.Snapshot, 0)

	keys, _ := self.client.Keys(self.ctx, "*").Result()

	// redis stack query engine, type = "none"
	ft, err := self.client.Do(self.ctx, "FT._LIST").Text()

	if err == nil {
		keys = append(keys, ft)
	}

	// keys
	for _, key := range keys {

		keyType, err := self.client.Type(self.ctx, key).Result()

		// Type failed
		if err != nil {
			return nil, errors.New("TYPE key failed.")
		}

		// special key
		fmap := map[string]func(string, *model.Snapshot) error{
			"hash": self.collectHash,
			// "geo" : collect geo,
			"stream": self.collectStream,
			// "none" : collect ft,
			// "TSDB-TYPE" : collect ts,
		}

		f, ok := fmap[keyType]

		// special key
		if ok {
			err := f(key, &snapshot)

			// failed
			if err != nil {
				return nil, err
			}
		
		// common key
		}else {

			keyToken := model.Token{
				Level : model.TOKEN_LEVEL_1,
				Text : key,
			}

			_, ok := snapshot[keyToken]

			if !ok {
				snapshot[keyToken] = make([]model.Token, 0)
			}
		}

	}

	return snapshot, nil
}

// private
func (self *Redi) collectHash(key string, snapshot *model.Snapshot) error {

	fields, err := self.client.HKeys(self.ctx, key).Result()

	// HKEYS failed
	if err != nil {
		return errors.New("collect hash failed.")
	}

	keyToken := model.Token{
		Level : model.TOKEN_LEVEL_1,
		Text : key,
	}

	_, ok := (*snapshot)[keyToken]

	if !ok {
		(*snapshot)[keyToken] = make([]model.Token, 0)
	}

	for _, field := range fields {

		fieldToken := model.Token{
			Level : model.TOKEN_LEVEL_2,
			Text : field,
		}

		(*snapshot)[keyToken] =append((*snapshot)[keyToken], fieldToken)
	}

	return nil

}

// private
func (self *Redi) collectStream(key string, snapshot *model.Snapshot) error {

	entries, err := self.client.XRange(self.ctx, key, "-", "+").Result()

	if err != nil {
		return errors.New("collect stream failed.")
	}

	keyToken := model.Token{
		Level : model.TOKEN_LEVEL_1,
		Text : key,
	}

	_, ok := (*snapshot)[keyToken]

	if !ok {
		(*snapshot)[keyToken] = make([]model.Token, 0)
	}

	for _, entry := range entries {

		for field := range entry.Values {

			fieldToken := model.Token{
				Level : model.TOKEN_LEVEL_2,
				Text : field,
			}
	
			(*snapshot)[keyToken] =append((*snapshot)[keyToken], fieldToken)
		}
	}

	return nil
}

// public
func (self *Redi) Stderr() string {

	return self.stderr.String()
}

// public
func (self *Redi) Debug() {

	log.Println("==== Redi ====")
	log.Printf("path: %s\n", self.path)
	log.Printf("args: %v\n", self.args)
}




