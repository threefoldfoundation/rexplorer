package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/threefoldfoundation/tfchain/pkg/config"
	"github.com/threefoldfoundation/tfchain/pkg/types"
)

func main() {
	cmd := new(Commands)
	cmd.RPCaddr = ":23112"
	cmd.RedisAddr, cmd.RedisDB = ":6379", 0
	cmd.BlockchainInfo = config.GetBlockchainInfo()

	// define commands
	cmdRoot := &cobra.Command{
		Use:   "rexplorer",
		Short: "start the rexplorer daemon",
		Args:  cobra.ExactArgs(0),
		PreRunE: func(*cobra.Command, []string) error {
			switch cmd.BlockchainInfo.NetworkName {
			case config.NetworkNameStandard:
				// Register the transaction controllers for all transaction versions
				// supported on the standard network
				types.RegisterTransactionTypesForStandardNetwork()
				// Forbid the usage of MultiSignatureCondition (and thus the multisig feature),
				// until the blockchain reached a height of 42000 blocks.
				types.RegisterBlockHeightLimitedMultiSignatureCondition(42000)
				// get chain constants and bootstrap peers
				cmd.ChainConstants = config.GetStandardnetGenesis()
				cmd.BootstrapPeers = config.GetStandardnetBootstrapPeers()
			case config.NetworkNameTest:
				// Register the transaction controllers for all transaction versions
				// supported on the test network
				types.RegisterTransactionTypesForTestNetwork()
				// Use our custom MultiSignatureCondition, just for testing purposes
				types.RegisterBlockHeightLimitedMultiSignatureCondition(0)
				// get chain constants and bootstrap peers
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
	cmdRoot.Flags().StringVar(
		&cmd.RedisAddr,
		"redis-addr",
		cmd.RedisAddr,
		"which (tcp) address the redis server listens on",
	)
	cmdRoot.Flags().StringVar(
		&cmd.ProfilingAddr,
		"profile-addr",
		cmd.ProfilingAddr,
		"enables profiling of this rexplorer instance as an http service",
	)
	cmdRoot.Flags().IntVar(
		&cmd.RedisDB,
		"redis-db",
		cmd.RedisDB,
		"which redis database slot to use",
	)
	cmdRoot.Flags().VarP(
		&cmd.EncodingType,
		"encoding", "e",
		"which encoding protocol to use, one of {json,msgp}",
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
