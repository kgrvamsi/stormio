package main

import (
	"stormstack.org/cloudio/controllers"
	"stormstack.org/cloudio/util"
	"fmt"
	seelog "github.com/cihub/seelog"
	"os"
	"runtime"
)

func main() {
	ncpu := runtime.NumCPU()
	runtime.GOMAXPROCS(runtime.NumCPU())
	util.LoadProperties()
	defer seelog.Flush()
	seelogconfig := util.GetString("path", "log-conf")
	rootPath := util.GetString("path", "config-root")

    logger, err := seelog.LoggerFromConfigAsFile(rootPath + "/" + seelogconfig)
	if err != nil {
		fmt.Printf("Failed to initialize log, %v\n", err.Error())
		os.Exit(2)
	}
	fmt.Printf("Loaded Log configuration from %s\n", rootPath + "/" + seelogconfig)
	seelog.ReplaceLogger(logger)
	host := util.GetString("server", "host")
	port := util.GetString("server", "port")
    fmt.Printf("Starting cloudio on %s:%s using %d CPUs", host, port, ncpu)
	seelog.Debugf("No of CPU's available :%d", ncpu)
	controllers.StartServer(host, port)
}
