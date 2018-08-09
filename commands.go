package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"runtime"

	"github.com/rivine/rivine/modules"
	"github.com/rivine/rivine/modules/consensus"
	"github.com/rivine/rivine/modules/gateway"
	"github.com/rivine/rivine/types"
	"github.com/spf13/cobra"
)

type Commands struct {
	BlockchainInfo types.BlockchainInfo
	ChainConstants types.ChainConstants
	BootstrapPeers []modules.NetAddress

	// the host:port to listen for RPC calls
	RPCaddr string

	// redis info
	RedisAddr string
	RedisDB   int

	// the parent directory where the individual module
	// directories will be created
	RootPersistentDir string
}

func (cmd *Commands) Root(_ *cobra.Command, args []string) (cmdErr error) {
	log.Println("starting rexplorer v" + version.String() + "...")

	// create database
	db, err := NewRedisDatabase(cmd.RedisAddr, cmd.RedisDB, cmd.BlockchainInfo)
	if err != nil {
		return fmt.Errorf("failed to create redis db client: %v", err)
	}

	// load all modules

	log.Println("loading rivine gateway module (1/3)...")
	gateway, err := gateway.New(
		cmd.RPCaddr, true, cmd.perDir("gateway"),
		cmd.BlockchainInfo, cmd.ChainConstants, cmd.BootstrapPeers)
	if err != nil {
		return fmt.Errorf("failed to create gateway module: %v", err)
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
		return fmt.Errorf("failed to create consensus module: %v", err)
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
		db, cs, cmd.BlockchainInfo, cmd.ChainConstants)
	if err != nil {
		return fmt.Errorf("failed to create explorer module: %v", err)
	}
	defer func() {
		log.Println("Closing explorer module...")
		err := explorer.Close()
		if err != nil {
			cmdErr = err
			log.Println("[ERROR] Closing explorer module resulted in an error: ", err)
		}
	}()

	// stop the server if a kill signal is caught
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill)

	log.Println("rexplorer is up and running...")

	// wait for server to be killed or the process to be done
	select {
	case <-sigChan:
		log.Println("\r\nCaught stop signal, quitting...")
	case <-context.Background().Done():
		log.Println("\r\nBackground context is done, quitting...")
	}
	log.Println("Goodbye!")
	return
}

func (cmd *Commands) perDir(module string) string {
	return path.Join(
		cmd.RootPersistentDir,
		cmd.BlockchainInfo.Name, cmd.BlockchainInfo.NetworkName,
		module)
}

func (cmd *Commands) Version(_ *cobra.Command, args []string) {
	fmt.Printf("Tool version            v%s\n", version)
	fmt.Printf("TFChain Daemon version  v%s\n", cmd.BlockchainInfo.ChainVersion.String())
	fmt.Printf("Rivine protocol version v%s\n", cmd.BlockchainInfo.ProtocolVersion.String())
	fmt.Println()
	fmt.Printf("Go Version   v%s\n", runtime.Version()[2:])
	fmt.Printf("GOOS         %s\n", runtime.GOOS)
	fmt.Printf("GOARCH       %s\n", runtime.GOARCH)

}
