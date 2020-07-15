/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package debug

import (
	"io/ioutil"
	"log"
	"os"
	"sync"
)

// if this directory is defined, the loggers will be instantiated
// (debug.log is automatically initialized)
const (
	debugLogDir = "PQ_DEBUG_LOG_DIRECTORY"
)

var (
	lock         sync.Mutex
	cleanupFuncs = make([]LoggerCleanupFunc, 0)
)

type LoggerCleanupFunc func()

func initNoopLogger() *log.Logger {
	return log.New(ioutil.Discard, "", log.Llongfile)
}

func registerCleanupFunc(cleanup func()) {
	lock.Lock()
	defer lock.Unlock()
	cleanupFuncs = append(cleanupFuncs, cleanup)
}

//// todo: manually call the teardown func for the debug/error so that we can use these in tests
func Teardown() {
	for _, f := range cleanupFuncs {
		f()
	}
}

func NewDebugLogger(logfileName string) *log.Logger {
	lgr := initNoopLogger()
	if l := os.Getenv(debugLogDir); l != "" {
		f, err := os.OpenFile(l+"/"+logfileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err == nil {
			lgr = log.New(f, "", log.LstdFlags)
			lgr.Printf("%v enabled\n", logfileName)
			registerCleanupFunc(func() {
				f.Close()
			})
		}
	}
	return lgr
}
