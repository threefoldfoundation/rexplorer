package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/threefoldfoundation/tfchain/pkg/config"
)

func main() {
	cmd := new(Commands)
	cmd.RPCaddr = ":23112"
	cmd.BlockchainInfo = config.GetBlockchainInfo()

	// define commands
	cmdRoot := &cobra.Command{
		Use:   "rexplorer",
		Short: "start the rexplorer daemon",
		Args:  cobra.ExactArgs(0),
		PreRunE: func(*cobra.Command, []string) error {
			switch cmd.BlockchainInfo.NetworkName {
			case "standard":
				cmd.ChainConstants = config.GetStandardnetGenesis()
				cmd.BootstrapPeers = config.GetStandardnetBootstrapPeers()
			case "testnet":
				cmd.ChainConstants = config.GetTestnetGenesis()
				cmd.BootstrapPeers = config.GetTestnetBootstrapPeers()
			default:
				return fmt.Errorf(
					"%q is an invalid network name, has to be one of {standard,testnet}",
					cmd.BlockchainInfo.NetworkName)
			}
			return nil
		},
		RunE: cmd.Root,
	}

	cmdVersion := &cobra.Command{
		Use:   "version",
		Short: "show versions of this tool",
		Args:  cobra.ExactArgs(0),
		Run:   cmd.Version,
	}

	// define command tree
	cmdRoot.AddCommand(
		cmdVersion,
	)

	// define flags
	cmdRoot.Flags().StringVarP(
		&cmd.RootPersistentDir,
		"persistent-directory", "d",
		cmd.RootPersistentDir,
		"location of the root diretory used to store persistent data of the daemon of "+cmd.BlockchainInfo.Name,
	)
	cmdRoot.Flags().StringVar(
		&cmd.RPCaddr,
		"rpc-addr",
		cmd.RPCaddr,
		"which port the gateway listens on",
	)
	cmdRoot.Flags().StringVarP(
		&cmd.BlockchainInfo.NetworkName,
		"network", "n",
		cmd.BlockchainInfo.NetworkName,
		"the name of the network to which the daemon connects, one of {standard,testnet}",
	)

	// execute logic
	if err := cmdRoot.Execute(); err != nil {
		os.Exit(1)
	}
}
