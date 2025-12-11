package db

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/fuxxcss/Etcd2fuzz/pkg/model"
	"github.com/fuxxcss/Etcd2fuzz/pkg/utils"
	"go.etcd.io/etcd/client/v3"
)

/*
 * Etcd Definition
 */

/*	Etcd Struct	*/
type Etcd struct {

	// proc ...
	path   string
	args   []string
	stderr bytes.Buffer
	proc   *exec.Cmd

	// runtime ...
	client *clientv3.Client
	ctx    context.Context
}

const (
	EtcdLineSep  string = "\n"
	EtcdTokenSep string = " "
)

/*
 * Etcd Functions
 */

func NewEtcd(feature utils.TargetFeature) *Etcd {

	// set global sep
	model.LineSep = EtcdLineSep
	model.TokenSep = EtcdTokenSep

	etcd := new(Etcd)

	// path, port
	var path, port string

	path = feature[utils.TARGET_PATH]
	port = feature[utils.TARGET_PORT]

	// cannot find path
	_, err := os.Stat(path)

	if err != nil {
		log.Fatalf("err: %s %v", path, err)
	}

	etcd.path = path

	// Etcd runtime
	etcd.client, err = clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:" + port},
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		log.Fatalf("err: %s %v", path, err)
	}

	etcd.ctx = context.Background()

	// check alive
	alive := etcd.CheckAlive()

	// already startup, shutdown first
	if alive {
		etcd.client.Do(etcd.ctx, "shutdown")
	}

	// Etcd args
	etcd.args = []string{
		// port
		"--port" + " " + port,
	}

	return etcd

}

/*
 * Etcd Interface
 */

// Etcd Struct
// public
func (self *Etcd) StartUp() error {

	self.proc = exec.Command(self.path, self.args...)
	self.proc.Stderr = &self.stderr

	// error
	err := self.proc.Start()

	// startup failed
	if err != nil {
		return err
	}

	// waiting Etcd startup
	fmt.Println("[*] waiting Etcd startup...")
	for {
		alive := self.CheckAlive()
		if alive {
			break
		}
	}

	// succeed
	fmt.Printf("[*] Etcd %s StartUp.\n", self.path)

	return nil
}

// public
func (self *Etcd) Restart() error {

	err := self.proc.Start()

	// restart failed
	if err != nil {
		return err
	}

	// waiting Etcd restart
	fmt.Println("[*] waiting Etcd restart...")
	for {
		alive := self.CheckAlive()
		if alive {
			break
		}
	}

	// db succeed
	fmt.Printf("[*] Etcd %v ReStart.\n", self.path)

	return nil
}

// public
func (self *Etcd) ShutDown() {

	// kill Etcd
	self.proc.Process.Kill()

}

// public
func (self *Etcd) CheckAlive() bool {

	// Etcd state
	_, err := self.client.Status(self.ctx, self.client.Endpoints()[0])

	// Etcd is not alive
	if err != nil {
		return false
	}

	return true
}

// public
func (self *Etcd) CleanUp() error {

	_, err := self.client.Delete(self.ctx, "", clientv3.WithPrefix())

	// flushall failed
	if err != nil {
		return err
	}

	return nil
}

// public
func (self *Etcd) Execute(tokens []string) (utils.TargetState, error) {

	// marshal string
	args := []interface{}{}

	for _, token := range tokens {
		args = append(args, token)
	}

	// state
	state := utils.STATE_OK

	_, err := self.client.Do(self.ctx, args...).Result()

	// execute failed
	if err != nil && err != Etcds.Nil {

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
func (self *Etcd) Collect() (model.Snapshot, error) {

	// snapshot
	snapshot := make(model.Snapshot, 0)

	keys, _ := self.client.Keys(self.ctx, "*").Result()

	// Etcds stack query engine, type = "none"
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
		} else {

			keyToken := model.Token{
				Level: model.TOKEN_LEVEL_1,
				Text:  key,
			}

			_, ok := snapshot[keyToken]

			if !ok {
				snapshot[keyToken] = make([]model.Token, 0)
			}
		}

	}

	return snapshot, nil
}

// public
func (self *Etcd) Stderr() string {

	return self.stderr.String()
}

// public
func (self *Etcd) Debug() {

	log.Println("==== Etcd ====")
	log.Printf("path: %s\n", self.path)
	log.Printf("args: %v\n", self.args)
}
