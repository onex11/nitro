// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/validator/server_common"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/util"
	deploycode "github.com/offchainlabs/nitro/deploy"
)

func main() {
	// read config file
	deployArg, err := ioutil.ReadFile("./rollup-config/deploy_config.json")
	if err != nil {
		panic(err)
	}
	var args map[string]string
	json.Unmarshal(deployArg, &args)
	deploy(args)
}

func deploy(args map[string]string) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlDebug)
	log.Root().SetHandler(glogger)
	log.Info("deploying rollup")

	ctx := context.Background()

	// parse from config file
	l1conn := args["l1conn"]
	l1ChainIdUint, _ := strconv.ParseUint(args["l1ChainIdUint"], 10, 64)
	ownerAddressString := args["ownerAddressString"]
	sequencerAddressString := args["sequencerAddressString"]
	maxDataSizeUint, _ := strconv.ParseUint(args["maxDataSizeUint"], 10, 64)
	wasmmoduleroot := args["wasmmoduleroot"]
	wasmrootpath := args["wasmrootpath"]
	l1privatekey := args["l1privatekey"]
	l2ChainName := args["l2chainname"]
	prod := args["prod"] == "true"

	// keep default value
	l1keystore := flag.String("l1keystore", "", "l1 private key store")
	l1passphrase := flag.String("l1passphrase", "passphrase", "l1 private key file passphrase")
	deployAccount := flag.String("l1DeployAccount", "", "l1 seq account to use (default is first account in keystore)")
	nativeTokenAddressString := flag.String("nativeTokenAddress", "0x0000000000000000000000000000000000000000", "address of the ERC20 token which is used as native L2 currency")
	// double check this
	loserEscrowAddressString := flag.String("loserEscrowAddress", "0x0000000000000000000000000000000000000000", "the address which half of challenge loser's funds accumulate at")
	l2ChainConfig := flag.String("l2chainconfig", "./rollup-config/l2_chain_config.json", "L2 chain config json file")
	outfile := flag.String("l1deployment", "./rollup-deployment/deploy.json", "deployment output json file")
	l2ChainInfo := flag.String("l2chaininfo", "./rollup-deployment/l2_chain_info.json", "L2 chain info output json file")
	authorizevalidators := flag.Uint64("authorizevalidators", 1, "Number of validators to preemptively authorize")
	txTimeout := flag.Duration("txtimeout", 10*time.Minute, "Timeout when waiting for a transaction to be included in a block")
	validatorWalletCreator := common.HexToAddress("0x06E341073b2749e0Bb9912461351f716DeCDa9b0")
	validatorUtils := common.HexToAddress("0xB11EB62DD2B352886A4530A9106fE427844D515f")
	rollupCreatorAddr := "0x06E341073b2749e0Bb9912461351f716DeCDa9b0"

	flag.Parse()
	l1ChainId := new(big.Int).SetUint64(l1ChainIdUint)
	maxDataSize := new(big.Int).SetUint64(maxDataSizeUint)

	if prod {
		if wasmmoduleroot == "" {
			panic("must specify wasm module root when launching prod chain")
		}
	}
	if l2ChainName == "" {
		panic("must specify l2 chain name")
	}

	wallet := genericconf.WalletConfig{
		Pathname:   *l1keystore,
		Account:    *deployAccount,
		Password:   *l1passphrase,
		PrivateKey: l1privatekey,
	}
	l1TransactionOpts, _, err := util.OpenWallet("l1", &wallet, l1ChainId)
	if err != nil {
		flag.Usage()
		log.Error("error reading keystore")
		panic(err)
	}

	l1client, err := ethclient.Dial(l1conn)
	if err != nil {
		flag.Usage()
		log.Error("error creating l1client")
		panic(err)
	}

	if !common.IsHexAddress(sequencerAddressString) && len(sequencerAddressString) > 0 {
		panic("specified sequencer address is invalid")
	}
	if !common.IsHexAddress(ownerAddressString) {
		panic("please specify a valid rollup owner address")
	}
	if prod && !common.IsHexAddress(*loserEscrowAddressString) {
		panic("please specify a valid loser escrow address")
	}

	sequencerAddress := common.HexToAddress(sequencerAddressString)
	ownerAddress := common.HexToAddress(ownerAddressString)
	loserEscrowAddress := common.HexToAddress(*loserEscrowAddressString)
	if sequencerAddress != (common.Address{}) && ownerAddress != l1TransactionOpts.From {
		panic("cannot specify sequencer address if owner is not deployer")
	}

	var moduleRoot common.Hash
	if wasmmoduleroot == "" {
		locator, err := server_common.NewMachineLocator(wasmrootpath)
		if err != nil {
			panic(err)
		}
		moduleRoot = locator.LatestWasmModuleRoot()
	} else {
		moduleRoot = common.HexToHash(wasmmoduleroot)
	}
	if moduleRoot == (common.Hash{}) {
		panic("wasmModuleRoot not found")
	}

	headerReaderConfig := headerreader.DefaultConfig
	headerReaderConfig.TxTimeout = *txTimeout

	chainConfigJson, err := os.ReadFile(*l2ChainConfig)
	if err != nil {
		panic(fmt.Errorf("failed to read l2 chain config file: %w", err))
	}
	var chainConfig params.ChainConfig
	err = json.Unmarshal(chainConfigJson, &chainConfig)
	if err != nil {
		panic(fmt.Errorf("failed to deserialize chain config: %w", err))
	}

	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, l1client)
	l1Reader, err := headerreader.New(ctx, l1client, func() *headerreader.Config { return &headerReaderConfig }, arbSys)
	if err != nil {
		panic(fmt.Errorf("failed to create header reader: %w", err))
	}
	l1Reader.Start(ctx)
	defer l1Reader.StopAndWait()

	nativeToken := common.HexToAddress(*nativeTokenAddressString)
	deployedAddresses, err := deploycode.DeployOnL1(
		ctx,
		l1Reader,
		l1TransactionOpts,
		sequencerAddress,
		*authorizevalidators,
		arbnode.GenerateRollupConfig(prod, moduleRoot, ownerAddress, &chainConfig, chainConfigJson, loserEscrowAddress),
		nativeToken,
		maxDataSize,
		*l1client,
		validatorWalletCreator,
		validatorUtils,
		rollupCreatorAddr,
	)
	if err != nil {
		flag.Usage()
		log.Error("error deploying on l1")
		panic(err)
	}
	deployData, err := json.Marshal(deployedAddresses)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*outfile, deployData, 0600); err != nil {
		panic(err)
	}
	parentChainIsArbitrum := l1Reader.IsParentChainArbitrum()
	chainsInfo := []chaininfo.ChainInfo{
		{
			ChainName:             l2ChainName,
			ParentChainId:         l1ChainId.Uint64(),
			ParentChainIsArbitrum: &parentChainIsArbitrum,
			ChainConfig:           &chainConfig,
			RollupAddresses:       deployedAddresses,
		},
	}
	chainsInfoJson, err := json.Marshal(chainsInfo)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*l2ChainInfo, chainsInfoJson, 0600); err != nil {
		panic(err)
	}
}
