/*
Copyright © 2021 OCTOPS.IO

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
package cmd

import (
	"context"
	"fmt"
	"github.com/Octops/gameserver-ingress-controller/internal/runtime"
	"github.com/Octops/gameserver-ingress-controller/pkg/controller"
	"github.com/Octops/gameserver-ingress-controller/pkg/handlers"
	"github.com/Octops/gameserver-ingress-controller/pkg/manager"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	masterURL  string
	kubeconfig string
	syncPeriod string
	port       int
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gameserver-ingress-controller",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		logger := runtime.NewLogger(true)

		clientConf, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			logger.Fatal(err)
		}

		duration, err := time.ParseDuration(syncPeriod)
		if err != nil {
			logger.WithError(err).Fatalf("error parsing sync-period flag: %s", syncPeriod)
		}

		mgr, err := manager.NewManager(clientConf, manager.Options{
			SyncPeriod: &duration,
			Port:       port,
		})

		logger.Info("creating GameServer controller")
		ctrl, err := controller.NewGameServerController(mgr, handlers.NewGameSeverEventHandler(clientConf), controller.Options{
			For: &agonesv1.GameServer{},
		})

		ctx, cancel := context.WithCancel(context.Background())
		runtime.SetupSignal(cancel)

		if err := ctrl.Start(ctx); err != nil {
			logger.Fatal(err)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gameserver-ingress-controller.yaml)")

	rootCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Set KUBECONFIG")
	rootCmd.Flags().StringVar(&masterURL, "master", "", "The addr of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	rootCmd.Flags().StringVar(&syncPeriod, "sync-period", "5s", "Set the minimum frequency at which watched resources are reconciled")
	rootCmd.Flags().IntVar(&port, "port", 30234, "Port used by the manager for webhooks")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".gameserver-ingress-controller" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".gameserver-ingress-controller")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
