package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"runtime"
	"sync"

	_ "net/http/pprof"

	"github.com/rivine/rivine/modules"
	"github.com/rivine/rivine/modules/consensus"
	"github.com/rivine/rivine/modules/gateway"
	"github.com/rivine/rivine/types"
	"github.com/threefoldfoundation/rexplorer/pkg/encoding"

	"github.com/spf13/cobra"
)

// Commands is the stateful object used as the central method-owning object
// for all Cobra (CLI) commands.
type Commands struct {
	BlockchainInfo types.BlockchainInfo
	ChainConstants types.ChainConstants
	BootstrapPeers []modules.NetAddress

	// the host:port to listen for RPC calls
	RPCaddr string

	// redis info
	RedisAddr string
	RedisDB   int

	// ProfilingAddr is optionally used to
	// enable the (std pprof) profiler and expose is as a HTTP interface
	ProfilingAddr string

	// encoding info
	EncodingType encoding.Type

	// the parent directory where the individual module
	// directories will be created
	RootPersistentDir string
}

// Root represents the root (`rexplorer`) command,
// starting a rexplorer daemon instance, running until the user intervenes.
func (cmd *Commands) Root(_ *cobra.Command, args []string) (cmdErr error) {
	log.Println("starting rexplorer v" + version.String() + "...")

	// optionally enable profiling and expose it over a HTTP interface
	if len(cmd.ProfilingAddr) > 0 {
		go func() {
			log.Println("profiling enabled, available on", cmd.ProfilingAddr)
			err := http.ListenAndServe(cmd.ProfilingAddr, http.DefaultServeMux)
			if err != nil {
				fmt.Println("[ERROR] profiler couldn't be started:", err)
			}
		}()
	}

	// create database
	db, err := NewRedisDatabase(cmd.RedisAddr, cmd.RedisDB, cmd.EncodingType, cmd.BlockchainInfo, cmd.ChainConstants)
	if err != nil {
		return fmt.Errorf("failed to create redis db client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// load all modules

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Println("loading rivine gateway module (1/3)...")
		gateway, err := gateway.New(
			cmd.RPCaddr, true, cmd.perDir("gateway"),
			cmd.BlockchainInfo, cmd.ChainConstants, cmd.BootstrapPeers)
		if err != nil {
			cmdErr = fmt.Errorf("failed to create gateway module: %v", err)
			log.Println("[ERROR] ", cmdErr)
			cancel()
			return
		}
		defer func() {
			log.Println("Closing gateway module...")
			err := gateway.Close()
			if err != nil {
				cmdErr = err
				log.Println("[ERROR] Closing gateway module resulted in an error: ", err)
			}
		}()

		log.Println("loading rivine consensus module (2/3)...")
		cs, err := consensus.New(
			gateway, true, cmd.perDir("consensus"),
			cmd.BlockchainInfo, cmd.ChainConstants)
		if err != nil {
			cmdErr = fmt.Errorf("failed to create consensus module: %v", err)
			log.Println("[ERROR] ", cmdErr)
			cancel()
			return
		}
		defer func() {
			log.Println("Closing consensus module...")
			err := cs.Close()
			if err != nil {
				cmdErr = err
				log.Println("[ERROR] Closing consensus module resulted in an error: ", err)
			}
		}()

		log.Println("loading internal explorer module (3/3)...")
		explorer, err := NewExplorer(
			db, cs, cmd.BlockchainInfo, cmd.ChainConstants, ctx.Done())
		if err != nil {
			cmdErr = fmt.Errorf("failed to create explorer module: %v", err)
			log.Println("[ERROR] ", cmdErr)
			cancel()
			return
		}
		defer func() {
			log.Println("Closing explorer module...")
			err := explorer.Close()
			if err != nil {
				cmdErr = err
				log.Println("[ERROR] Closing explorer module resulted in an error: ", err)
			}
		}()

		log.Println("rexplorer is up and running...")

		// wait until done
		<-ctx.Done()
	}()

	// stop the server if a kill signal is caught
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill)

	// wait for server to be killed or the process to be done
	select {
	case <-sigChan:
		log.Println("\r\nCaught stop signal, quitting...")
	case <-context.Background().Done():
		log.Println("\r\ncontext is done, quitting...")
	}

	cancel()
	wg.Wait()

	log.Println("Goodbye!")
	return
}

func (cmd *Commands) perDir(module string) string {
	return path.Join(
		cmd.RootPersistentDir,
		cmd.BlockchainInfo.Name, cmd.BlockchainInfo.NetworkName,
		module)
}

// Version represents the version (`rexplorer version`) command,
// returning the version of the tool, dependencies and Go,
// as well as the OS and Arch type.
func (cmd *Commands) Version(_ *cobra.Command, args []string) {
	fmt.Printf("Tool version            v%s\n", version.String())
	fmt.Printf("TFChain Daemon version  v%s\n", cmd.BlockchainInfo.ChainVersion.String())
	fmt.Printf("Rivine protocol version v%s\n", cmd.BlockchainInfo.ProtocolVersion.String())
	fmt.Println()
	fmt.Printf("Go Version   v%s\n", runtime.Version()[2:])
	fmt.Printf("GOOS         %s\n", runtime.GOOS)
	fmt.Printf("GOARCH       %s\n", runtime.GOARCH)

}
