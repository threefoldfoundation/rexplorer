package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"runtime"
	"sync"

	_ "net/http/pprof"

	"github.com/threefoldfoundation/rexplorer/pkg/encoding"
	"github.com/threefoldfoundation/rexplorer/pkg/types"
	tfconsensus "github.com/threefoldfoundation/tfchain/extensions/tfchain/consensus"
	tftypes "github.com/threefoldfoundation/tfchain/pkg/types"

	"github.com/threefoldfoundation/tfchain/extensions/threebot"
	"github.com/threefoldfoundation/tfchain/pkg/config"

	erc20 "github.com/threefoldtech/rivine-extension-erc20"
	erc20types "github.com/threefoldtech/rivine-extension-erc20/types"
	"github.com/threefoldtech/rivine/extensions/authcointx"
	"github.com/threefoldtech/rivine/extensions/minting"
	"github.com/threefoldtech/rivine/modules"
	"github.com/threefoldtech/rivine/modules/consensus"
	"github.com/threefoldtech/rivine/modules/gateway"
	rivinetypes "github.com/threefoldtech/rivine/types"

	"github.com/spf13/cobra"
)

// Commands is the stateful object used as the central method-owning object
// for all Cobra (CLI) commands.
type Commands struct {
	BlockchainInfo rivinetypes.BlockchainInfo
	ChainConstants rivinetypes.ChainConstants
	BootstrapPeers []modules.NetAddress

	DaemonNetworkConfig  config.DaemonNetworkConfig
	GenesisAuthCondition rivinetypes.UnlockConditionProxy

	Validators       []modules.TransactionValidationFunction
	MappedValidators map[rivinetypes.TransactionVersion][]modules.TransactionValidationFunction
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

	// the outputs which description match will be stored
	// in wallet values, even when unlocked.
	DescriptionFilterSet types.DescriptionFilterSet

	// YesToAll is a bool property that allows you to answer any
	// question that would otherwise require answering manually via the STDIN
	YesToAll bool

	// the parent directory where the individual module
	// directories will be created
	RootPersistentDir string
}

// Root represents the root (`rexplorer`) command,
// starting a rexplorer daemon instance, running until the user intervenes.
func (cmd *Commands) Root(_ *cobra.Command, args []string) (cmdErr error) {
	log.Println("starting rexplorer v" + version.String() + "...")

	log.Println("loading network config, registering types and loading rivine transaction db (0/3)...")
	switch cmd.BlockchainInfo.NetworkName {
	case config.NetworkNameStandard:
		// get chain constants and bootstrap peers
		cmd.ChainConstants = config.GetStandardnetGenesis()
		cmd.DaemonNetworkConfig = config.GetStandardDaemonNetworkConfig()
		cmd.GenesisAuthCondition = config.GetStandardnetGenesisAuthCoinCondition()
		cmd.Validators = tfconsensus.GetStandardTransactionValidators()
		cmd.MappedValidators = tfconsensus.GetStandardTransactionVersionMappedValidators()

		if len(cmd.BootstrapPeers) == 0 {
			cmd.BootstrapPeers = config.GetStandardnetBootstrapPeers()
		}

	case config.NetworkNameTest:
		// get chain constants and bootstrap peers
		cmd.ChainConstants = config.GetTestnetGenesis()
		cmd.DaemonNetworkConfig = config.GetTestnetDaemonNetworkConfig()
		cmd.GenesisAuthCondition = config.GetTestnetGenesisAuthCoinCondition()
		cmd.Validators = tfconsensus.GetTestnetTransactionValidators()
		cmd.MappedValidators = tfconsensus.GetTestnetTransactionVersionMappedValidators()
		if len(cmd.BootstrapPeers) == 0 {
			cmd.BootstrapPeers = config.GetTestnetBootstrapPeers()
		}

	case config.NetworkNameDev:
		// get chain constants and bootstrap peers
		cmd.ChainConstants = config.GetDevnetGenesis()
		cmd.DaemonNetworkConfig = config.GetDevnetDaemonNetworkConfig()
		cmd.GenesisAuthCondition = config.GetDevnetGenesisAuthCoinCondition()
		cmd.Validators = tfconsensus.GetDevnetTransactionValidators()
		cmd.MappedValidators = tfconsensus.GetDevnetTransactionVersionMappedValidators()
		if len(cmd.BootstrapPeers) == 0 {
			return errors.New("no bootstrap peers are defined while this is required for devnet (using the --bootstrap-peer flag)")
		}

	default:
		return fmt.Errorf(
			"%q is an invalid network name, has to be one of {standard,testnet,devnet}",
			cmd.BlockchainInfo.NetworkName)
	}

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
	db, err := NewRedisDatabase(
		cmd.RedisAddr, cmd.RedisDB,
		cmd.EncodingType, cmd.BlockchainInfo, cmd.ChainConstants,
		cmd.DescriptionFilterSet, cmd.YesToAll)
	if err != nil {
		return fmt.Errorf("failed to create redis db client "+
			"(if you use an existing Redis DB ensure you are using the same flags as used previously): %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// load all modules

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Println("loading rivine gateway module (1/3)...")
		gateway, err := gateway.New(
			cmd.RPCaddr, true, 1, cmd.perDir("gateway"),
			cmd.BlockchainInfo, cmd.ChainConstants, cmd.BootstrapPeers, false)
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

		var mintingPlugin *minting.Plugin
		var threebotPlugin *threebot.Plugin
		var erc20TxValidator erc20types.ERC20TransactionValidator
		var erc20Plugin *erc20.Plugin
		var authCoinTxPlugin *authcointx.Plugin

		log.Println("loading rivine consensus module (2/3)...")
		cs, err := consensus.New(
			gateway, true, cmd.perDir("consensus"),
			cmd.BlockchainInfo, cmd.ChainConstants, false, "")
		if err != nil {
			cmdErr = fmt.Errorf("failed to create consensus module: %v", err)
			log.Println("[ERROR] ", cmdErr)
			cancel()
			return
		}

		cs.SetTransactionValidators(cmd.Validators...)
		for txVersion, validators := range cmd.MappedValidators {
			cs.SetTransactionVersionMappedValidators(txVersion, validators...)
		}
		defer func() {
			log.Println("Closing consensus module...")
			err := cs.Close()
			if err != nil {
				cmdErr = err
				log.Println("[ERROR] Closing consensus module resulted in an error: ", err)
			}
		}()

		// create the minting extension plugin
		mintingPlugin = minting.NewMintingPlugin(
			cmd.DaemonNetworkConfig.GenesisMintingCondition,
			tftypes.TransactionVersionMinterDefinition,
			tftypes.TransactionVersionCoinCreation,
			&minting.PluginOptions{
				UseLegacySiaEncoding: true,
				RequireMinerFees:     true,
			},
		)

		// 3Bot and ERC20 is not yet to be used on network standard
		if cmd.BlockchainInfo.NetworkName != config.NetworkNameStandard {
			// create the 3Bot plugin
			var tbPluginOpts *threebot.PluginOptions
			if cmd.BlockchainInfo.NetworkName == config.NetworkNameTest {
				tbPluginOpts = &threebot.PluginOptions{ // TODO: remove this hack once possible (e.g. a testnet network reset)
					HackMinimumBlockHeightSinceDoubleRegistrationsAreForbidden: 350000,
				}
			}
			threebotPlugin = threebot.NewPlugin(
				cmd.DaemonNetworkConfig.FoundationPoolAddress,
				cmd.ChainConstants.CurrencyUnits.OneCoin,
				tbPluginOpts,
			)

			// create a non-validating ERC20 Tx Validator
			erc20TxValidator = erc20types.NopERC20TransactionValidator{}

			// create the ERC20 plugin
			erc20Plugin = erc20.NewPlugin(
				cmd.DaemonNetworkConfig.ERC20FeePoolAddress,
				cmd.ChainConstants.CurrencyUnits.OneCoin,
				erc20TxValidator,
				erc20types.TransactionVersions{
					ERC20Conversion:          tftypes.TransactionVersionERC20Conversion,
					ERC20AddressRegistration: tftypes.TransactionVersionERC20AddressRegistration,
					ERC20CoinCreation:        tftypes.TransactionVersionERC20CoinCreation,
				},
			)

			// register the ERC20 Plugin
			err = cs.RegisterPlugin(ctx, "erc20", erc20Plugin)
			if err != nil {
				cmdErr = fmt.Errorf("failed to register the ERC20 extension: %v", err)
				log.Println("[ERROR] ", cmdErr)
				err = erc20Plugin.Close() //make sure any resources are released
				if err != nil {
					fmt.Println("Error during closing of the erc20Plugin:", err)
				}
				cancel()
				return
			}
			// register the Threebot Plugin
			err = cs.RegisterPlugin(ctx, "threebot", threebotPlugin)
			if err != nil {
				cmdErr = fmt.Errorf("failed to register the threebot extension: %v", err)
				log.Println("[ERROR] ", cmdErr)
				err = threebotPlugin.Close() //make sure any resources are released
				if err != nil {
					fmt.Println("Error during closing of the threebotPlugin:", err)
				}
				cancel()
				return
			}
		}

		// register the Minting Plugin
		err = cs.RegisterPlugin(ctx, "minting", mintingPlugin)
		if err != nil {
			cmdErr = fmt.Errorf("failed to register the threebot extension: %v", err)
			log.Println("[ERROR] ", cmdErr)
			err = mintingPlugin.Close() //make sure any resources are released
			if err != nil {
				fmt.Println("Error during closing of the mintingPlugin :", err)
			}
			cancel()
			return
		}

		// create the auth coin tx plugin
		// > NOTE: this also overwrites the standard tx controllers!!!!
		authCoinTxPlugin = authcointx.NewPlugin(
			cmd.GenesisAuthCondition,
			tftypes.TransactionVersionAuthAddressUpdate,
			tftypes.TransactionVersionAuthConditionUpdate,
			&authcointx.PluginOpts{
				UnauthorizedCoinTransactionExceptionCallback: func(tx modules.ConsensusTransaction, dedupAddresses []rivinetypes.UnlockHash, ctx rivinetypes.TransactionValidationContext) (bool, error) {
					if tx.Version != rivinetypes.TransactionVersionZero && tx.Version != rivinetypes.TransactionVersionOne {
						return false, nil
					}
					return (len(dedupAddresses) == 1 && len(tx.CoinOutputs) <= 2), nil
				},
				RequireMinerFees:    true,
				AuthorizedByDefault: true,
			},
		)

		// register the AuthCoin extension plugin
		err = cs.RegisterPlugin(ctx, "authcointx", authCoinTxPlugin)
		if err != nil {
			cmdErr = fmt.Errorf("failed to register the auth coin tx extension: %v", err)
			log.Println("[ERROR] ", cmdErr)
			err = authCoinTxPlugin.Close() //make sure any resources are released
			if err != nil {
				fmt.Println("Error during closing of the authCoinTxPlugin :", err)
			}
			cancel()
			return
		}

		log.Println("loading internal explorer module (3/3)...")
		explorer, err := NewExplorer(
			db, cs, cmd.BlockchainInfo, cmd.ChainConstants, mintingPlugin, ctx.Done())
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

		cs.Start()

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
		log.Println("Caught stop signal, quitting...")
	case <-ctx.Done():
		log.Println("context is done, quitting...")
	}

	cancel()
	wg.Wait()

	log.Println("Goodbye!")
	return
}

func (cmd *Commands) rootPerDir() string {
	return path.Join(
		cmd.RootPersistentDir,
		cmd.BlockchainInfo.Name, cmd.BlockchainInfo.NetworkName)
}

func (cmd *Commands) perDir(module string) string {
	return path.Join(cmd.rootPerDir(), module)
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
