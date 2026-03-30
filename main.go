package main

import (
	"os"
	"runtime"

	"github.com/kyverno/reports-server/pkg/app"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
	cmd := app.NewPolicyServer(genericapiserver.SetupSignalHandler())
	if err := cmd.Execute(); err != nil {
		klog.Errorf("failed to execute: %s", err.Error())
		os.Exit(1)
	}
}
