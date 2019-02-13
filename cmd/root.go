// Copyright (c) 2019 Blake Rouse.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS
// OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
// IN THE SOFTWARE.

package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/blakerouse/external-ambassador/sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	// Load all auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var kubeFile string
var serviceName string
var serviceNamespace string

// Base command of the application
var rootCmd = &cobra.Command{
	Use:   "external-ambassador",
	Short: "Connector between ambassador and Kubernetes external-dns",
	Long: `Watches for host entries in ambassador annotations automatically
updating the external-dns annotations on the ambassador service.

The change in the external-dns annotation on the ambassador service will
cause external-dns to point the DNS host entry at the external IP address of
the ambassador service.`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var config *rest.Config

		if kubeFile == "" {
			config, err = rest.InClusterConfig()
			if err != nil {
				panic(err.Error())
			}
		} else {
			config, err = clientcmd.BuildConfigFromFlags("", kubeFile)
			if err != nil {
				panic(err.Error())
			}
		}

		watcher, err := sync.NewWatcher(config, serviceNamespace, serviceName)
		if err != nil {
			panic(err.Error())
		}
		if err := watcher.Start(); err != nil {
			panic(err.Error())
		}
		for {
			time.Sleep(time.Second)
		}
		_ = watcher.Stop()
	},
}

// Executes the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initLogger)

	rootCmd.PersistentFlags().BoolP("debug", "d", false, "enable debug output")
	rootCmd.PersistentFlags().StringVarP(&kubeFile, "kubeconfig", "c", "", "kubeconfig file to use")
	rootCmd.PersistentFlags().StringVarP(&serviceNamespace, "namespace", "n", "default", "namespace of the ambassador service")
	rootCmd.PersistentFlags().StringVarP(&serviceName, "service", "s", "ambassador", "name of the ambassador service")
}

// initLogger initialized the logger level.
func initLogger() {
	debug, err := rootCmd.PersistentFlags().GetBool("debug")
	if err != nil {
		// should not happen
		panic(err)
	}
	if debug {
		log.SetLevel(log.DebugLevel)
	}
}
