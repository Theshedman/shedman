package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/api"
	"github.com/theshedman/shedman/pkg/core"
)

var apiAddr string

// ApiCmd starts the shedman API server.
var ApiCmd = &cobra.Command{
	Use:   "api",
	Short: "Start the shedman API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunAPI(engine, apiAddr)
	},
}

type apiServer interface {
	Serve(addr string) error
}

var apiServerFactory = func(engine *core.Engine) apiServer {
	return api.NewServer(engine)
}

// RunAPI starts the API server.
func RunAPI(engine *core.Engine, addr string) error {
	if engine == nil {
		return fmt.Errorf("engine not available")
	}
	return apiServerFactory(engine).Serve(addr)
}

func init() {
	ApiCmd.Flags().StringVar(&apiAddr, "addr", "127.0.0.1:7337", "API listen address")
}
