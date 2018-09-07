package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/threefoldfoundation/rexplorer/pkg/rflag"
	"github.com/threefoldfoundation/tfchain/pkg/config"
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
		Long: `start the rexplorer daemon
		
(*) The following kind of glob expressions can
be made for the optional description filters:

  - Multiple Characters Wildcard: *
	example: 'bar:*' // matches 'bar:', 'bar: foo', 'bar:ok', ...

  - Single Character Wildcard: ?
	example: 'h?' // matches 'ho', 'ha', 'hi', 'hh', ...

  - Allow any character from a given character list: [xy]
	example: 'b[aoi]t' // matches 'bot', 'bit', 'bat'

  - Allow any character from a given character range: [x-y]
	example: 'a[a-c]a' // matches 'aaa', 'aba', 'aca'

  - Allow any character except a given character list: [!xy]
	example: 'a[!ac]ba' // does not match 'acba',
	                    // but does match 'abba', 'adba', ...

  - Allow any character except a character within a given character range: [!x-y]
	example: '[!a-f]oo' // does not match 'foo', 'boo', ...
	                    // but does match 'woo', 'zoo', 5oo, ...
`,
		Args: cobra.ExactArgs(0),
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
		"which encoding protocol to use, one of {json,msgp,protobuf}",
	)
	rflag.DescriptionFilterSetFlagVarP(
		cmdRoot.Flags(),
		&cmd.DescriptionFilterSet,
		"filter", "f",
		"list unlocked outputs in wallets if the description of output matches any of the unique glob (*) filters",
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
