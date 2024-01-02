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

type NodeConfig struct {
	Chain       Chain       `json:"chain"`
	ParentChain ParentChain `json:"parent-chain"`
	HTTP        HTTP        `json:"http"`
	Node        Node        `json:"node"`
}

type Dangerous struct {
	NoCoordinator bool `json:"no-coordinator"`
}
type Sequencer struct {
	MaxTxDataSize int       `json:"max-tx-data-size"`
	Enable        bool      `json:"enable"`
	Dangerous     Dangerous `json:"dangerous"`
	MaxBlockSpeed string    `json:"max-block-speed"`
}
type DelayedSequencer struct {
	Enable bool `json:"enable"`
}
type ParentChainWallet struct {
	PrivateKey string `json:"private-key"`
}
type BatchPoster struct {
	MaxSize           int               `json:"max-size"`
	Enable            bool              `json:"enable"`
	ParentChainWallet ParentChainWallet `json:"parent-chain-wallet"`
}
type Staker struct {
	Enable            bool              `json:"enable"`
	Strategy          string            `json:"strategy"`
	ParentChainWallet ParentChainWallet `json:"parent-chain-wallet"`
}
type Caching struct {
	Archive bool `json:"archive"`
}
type RestAggregator struct {
	Enable bool   `json:"enable"`
	Urls   string `json:"urls"`
}
type RPCAggregator struct {
	Enable        bool   `json:"enable"`
	AssumedHonest int    `json:"assumed-honest"`
	Backends      string `json:"backends"`
}
type DataAvailability struct {
	Enable                bool           `json:"enable"`
	SequencerInboxAddress string         `json:"sequencer-inbox-address"`
	ParentChainNodeURL    string         `json:"parent-chain-node-url"`
	RestAggregator        RestAggregator `json:"rest-aggregator"`
	RPCAggregator         RPCAggregator  `json:"rpc-aggregator"`
}
type Node struct {
	ForwardingTarget string           `json:"forwarding-target"`
	Sequencer        Sequencer        `json:"sequencer"`
	DelayedSequencer DelayedSequencer `json:"delayed-sequencer"`
	BatchPoster      BatchPoster      `json:"batch-poster"`
	Staker           Staker           `json:"staker"`
	Caching          Caching          `json:"caching"`
	DataAvailability DataAvailability `json:"data-availability"`
}

type Clique struct {
	Period int `json:"period"`
	Epoch  int `json:"epoch"`
}

type Arbitrum struct {
	EnableArbOS               bool   `json:"EnableArbOS"`
	AllowDebugPrecompiles     bool   `json:"AllowDebugPrecompiles"`
	DataAvailabilityCommittee bool   `json:"DataAvailabilityCommittee"`
	InitialArbOSVersion       int    `json:"InitialArbOSVersion"`
	InitialChainOwner         string `json:"InitialChainOwner"`
	GenesisBlockNum           int    `json:"GenesisBlockNum"`
}

type ChainConfig struct {
	ChainID             int64    `json:"chainId"`
	HomesteadBlock      int      `json:"homesteadBlock"`
	DaoForkBlock        any      `json:"daoForkBlock"`
	DaoForkSupport      bool     `json:"daoForkSupport"`
	Eip150Block         int      `json:"eip150Block"`
	Eip150Hash          string   `json:"eip150Hash"`
	Eip155Block         int      `json:"eip155Block"`
	Eip158Block         int      `json:"eip158Block"`
	ByzantiumBlock      int      `json:"byzantiumBlock"`
	ConstantinopleBlock int      `json:"constantinopleBlock"`
	PetersburgBlock     int      `json:"petersburgBlock"`
	IstanbulBlock       int      `json:"istanbulBlock"`
	MuirGlacierBlock    int      `json:"muirGlacierBlock"`
	BerlinBlock         int      `json:"berlinBlock"`
	LondonBlock         int      `json:"londonBlock"`
	Clique              Clique   `json:"clique"`
	Arbitrum            Arbitrum `json:"arbitrum"`
}

type Rollup struct {
	Bridge                 string `json:"bridge"`
	Inbox                  string `json:"inbox"`
	SequencerInbox         string `json:"sequencer-inbox"`
	Rollup                 string `json:"rollup"`
	ValidatorUtils         string `json:"validator-utils"`
	ValidatorWalletCreator string `json:"validator-wallet-creator"`
	DeployedAt             int    `json:"deployed-at"`
}

type Info struct {
	ChainID       int64       `json:"chain-id"`
	ParentChainID int         `json:"parent-chain-id"`
	ChainName     string      `json:"chain-name"`
	ChainConfig   ChainConfig `json:"chain-config"`
	Rollup        Rollup      `json:"rollup"`
}

type Chain struct {
	InfoJSON string `json:"info-json"`
	Name     string `json:"name"`
}

type Connection struct {
	URL string `json:"url"`
}

type ParentChain struct {
	Connection Connection `json:"connection"`
}

type HTTP struct {
	Addr       string   `json:"addr"`
	Port       int      `json:"port"`
	Vhosts     string   `json:"vhosts"`
	Corsdomain string   `json:"corsdomain"`
	API        []string `json:"api"`
}

type InitConfig struct {
	Force           bool          `koanf:"force"`
	Url             string        `koanf:"url"`
	DownloadPath    string        `koanf:"download-path"`
	DownloadPoll    time.Duration `koanf:"download-poll"`
	DevInit         bool          `koanf:"dev-init"`
	DevInitAddress  string        `koanf:"dev-init-address"`
	DevInitBlockNum uint64        `koanf:"dev-init-blocknum"`
	Empty           bool          `koanf:"empty"`
	AccountsPerSync uint          `koanf:"accounts-per-sync"`
	ImportFile      string        `koanf:"import-file"`
	ThenQuit        bool          `koanf:"then-quit"`
	Prune           string        `koanf:"prune"`
	PruneBloomSize  uint64        `koanf:"prune-bloom-size"`
	ResetToMessage  int64         `koanf:"reset-to-message"`
}

type OrbitSetupScriptConfig struct {
	ChainId                    uint64 `json:"chainId"`
	ChainName                  string `json:"chainName"`
	MinL2BaseFee               uint64 `json:"minL2BaseFee"`
	ParentChainId              uint64 `json:"parentChainId"`
	ParentChainNodeUrl         string `json:"parent-chain-node-url"`
	BatchPoster                string `json:"batchPoster"`
	Staker                     string `json:"staker"`
	Outbox                     string `json:"outbox"`
	AdminProxy                 string `json:"adminProxy"`
	NetworkFeeReceiver         string `json:"networkFeeReceiver"`
	InfrastructureFeeCollector string `json:"infrastructureFeeCollector"`
	ChainOwner                 string `json:"chainOwner"`
	Bridge                     string `json:"bridge"`
	Inbox                      string `json:"inbox"`
	SequencerInbox             string `json:"sequencerInbox"`
	Rollup                     string `json:"rollup"`
	NativeToken                string `json:"nativeToken"`
	UpgradeExecutor            string `json:"upgradeExecutor"`
	Utils                      string `json:"utils"`
	ValidatorWalletCreator     string `json:"validatorWalletCreator"`
	DeployedAtBlockNumber      int64  `json:"deployedAtBlockNumber"`
}

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
	networkFeeReceiver := args["networkFeeReceiver"]
	infrastructureFeeCollector := args["infrastructureFeeCollector"]
	sequencerAddressString := args["sequencerAddressString"]
	stakerAddressString := args["stakerAddressString"]
	maxDataSizeUint, _ := strconv.ParseUint(args["maxDataSizeUint"], 10, 64)
	wasmmoduleroot := args["wasmmoduleroot"]
	wasmrootpath := args["wasmrootpath"]
	l1privatekey := args["l1privatekey"]
	batcherPrivateKey := args["batcherPrivateKey"]
	stakePrivateKey := args["stakePrivateKey"]
	l2ChainName := args["l2chainname"]
	prod := args["prod"] == "true"
	minimumstake := args["minimumstake"]
	challengePeriodBlocks := args["challengePeriodBlocks"]
	nativeTokenAddressString := args["nativeTokenAddressString"]
	rollupCreatorAddr := args["rollupCreatorAddr"]
	stakeTokenAddressString := args["stakeTokenAddressString"]
	anyTrustMode := args["anyTrustMode"] == "true"

	// keep default value
	l1keystore := flag.String("l1keystore", "", "l1 private key store")
	l1passphrase := flag.String("l1passphrase", "passphrase", "l1 private key file passphrase")
	deployAccount := flag.String("l1DeployAccount", "", "l1 seq account to use (default is first account in keystore)")
	// double check this
	loserEscrowAddressString := flag.String("loserEscrowAddress", "0x0000000000000000000000000000000000000000", "the address which half of challenge loser's funds accumulate at")
	l2ChainConfig := flag.String("l2chainconfig", "./rollup-config/l2_chain_config.json", "L2 chain config json file")
	outfile := flag.String("l1deployment", "./rollup-deployment/deploy.json", "deployment output json file")
	l2ChainInfo := flag.String("l2chaininfo", "./rollup-deployment/l2_chain_info.json", "L2 chain info output json file")
	orbitSetupScriptConfigInfo := flag.String("orbitSetupScriptConfigInfo", "./rollup-deployment/orbitSetupScriptConfig.json", "L2 chain info output json file")
	nodeConfigInfo := flag.String("nodeconfig", "./rollup-deployment/nodeConfig.json", "L2 chain info output json file")
	authorizevalidators := flag.Uint64("authorizevalidators", 1, "Number of validators to preemptively authorize")
	txTimeout := flag.Duration("txtimeout", 10*time.Minute, "Timeout when waiting for a transaction to be included in a block")
	validatorWalletCreator := common.HexToAddress("0x06E341073b2749e0Bb9912461351f716DeCDa9b0")
	validatorUtils := common.HexToAddress("0xB11EB62DD2B352886A4530A9106fE427844D515f")

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
	stakerAddress := common.HexToAddress(stakerAddressString)
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

	nativeToken := common.HexToAddress(nativeTokenAddressString)

	deployedAddresses, err := deploycode.DeployOnL1(
		ctx,
		l1Reader,
		l1TransactionOpts,
		sequencerAddress,
		stakerAddress,
		*authorizevalidators,
		arbnode.GenerateRollupConfig(prod, moduleRoot, ownerAddress, &chainConfig, chainConfigJson, loserEscrowAddress, minimumstake, challengePeriodBlocks, stakeTokenAddressString),
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
	orbitSetupScriptConfig := &OrbitSetupScriptConfig{
		ChainId:                    chainConfig.ChainID.Uint64(),
		ChainName:                  l2ChainName,
		MinL2BaseFee:               100000000,
		ParentChainId:              l1ChainIdUint,
		ParentChainNodeUrl:         l1conn,
		BatchPoster:                sequencerAddressString,
		Staker:                     stakerAddressString,
		Outbox:                     deployedAddresses.OutBox.Hex(),
		AdminProxy:                 deployedAddresses.AdminProxy.Hex(),
		NetworkFeeReceiver:         networkFeeReceiver,
		InfrastructureFeeCollector: infrastructureFeeCollector,
		ChainOwner:                 ownerAddressString,
		Bridge:                     deployedAddresses.Bridge.Hex(),
		Inbox:                      deployedAddresses.Inbox.Hex(),
		SequencerInbox:             deployedAddresses.SequencerInbox.Hex(),
		Rollup:                     deployedAddresses.Rollup.Hex(),
		NativeToken:                nativeTokenAddressString,
		UpgradeExecutor:            deployedAddresses.UpgradeExecutor.Hex(),
		Utils:                      validatorUtils.Hex(),
		ValidatorWalletCreator:     validatorUtils.Hex(),
		DeployedAtBlockNumber:      int64(deployedAddresses.DeployedAt),
	}
	orbitSetupScriptConfigJson, err := json.Marshal(orbitSetupScriptConfig)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*orbitSetupScriptConfigInfo, orbitSetupScriptConfigJson, 0600); err != nil {
		panic(err)
	}

	clique := Clique{
		Period: 0,
		Epoch:  0,
	}

	arbitrum := Arbitrum{
		EnableArbOS:               true,
		AllowDebugPrecompiles:     false,
		DataAvailabilityCommittee: anyTrustMode,
		InitialArbOSVersion:       10,
		InitialChainOwner:         ownerAddressString,
		GenesisBlockNum:           0,
	}

	chainCfg := ChainConfig{
		ChainID:             chainConfig.ChainID.Int64(),
		HomesteadBlock:      0,
		DaoForkBlock:        nil,
		DaoForkSupport:      true,
		Eip150Block:         0,
		Eip150Hash:          "0x0000000000000000000000000000000000000000000000000000000000000000",
		Eip155Block:         0,
		Eip158Block:         0,
		ByzantiumBlock:      0,
		ConstantinopleBlock: 0,
		PetersburgBlock:     0,
		IstanbulBlock:       0,
		MuirGlacierBlock:    0,
		BerlinBlock:         0,
		LondonBlock:         0,
		Clique:              clique,
		Arbitrum:            arbitrum,
	}

	rollup := Rollup{
		Bridge:                 deployedAddresses.Bridge.Hex(),
		Inbox:                  deployedAddresses.Inbox.Hex(),
		SequencerInbox:         deployedAddresses.SequencerInbox.Hex(),
		Rollup:                 deployedAddresses.Rollup.Hex(),
		ValidatorUtils:         validatorUtils.Hex(),
		ValidatorWalletCreator: validatorUtils.Hex(),
		DeployedAt:             int(deployedAddresses.DeployedAt),
	}

	info := []Info{
		{
			ChainID:       chainConfig.ChainID.Int64(),
			ParentChainID: int(l1ChainIdUint),
			ChainName:     l2ChainName,
			ChainConfig:   chainCfg,
			Rollup:        rollup,
		},
	}

	infoJson, _ := json.Marshal(info)
	chain := Chain{
		InfoJSON: string(infoJson),
		Name:     l2ChainName,
	}

	connection := Connection{
		URL: l1conn,
	}
	parentChain := ParentChain{
		Connection: connection,
	}

	http := HTTP{
		Addr:       "0.0.0.0",
		Port:       8449,
		Vhosts:     "*",
		Corsdomain: "*",
		API: []string{"eth",
			"net",
			"web3",
			"arb",
			"debug"},
	}

	dangerous := Dangerous{
		NoCoordinator: true,
	}

	sequencer := Sequencer{
		MaxTxDataSize: int(maxDataSize.Int64()),
		Enable:        true,
		Dangerous:     dangerous,
		MaxBlockSpeed: "250ms",
	}

	delayedSequencer := DelayedSequencer{
		Enable: true,
	}

	batcherWallet := ParentChainWallet{
		PrivateKey: batcherPrivateKey,
	}

	batchPoster := BatchPoster{
		MaxSize:           90000,
		Enable:            true,
		ParentChainWallet: batcherWallet,
	}

	stakerWallet := ParentChainWallet{
		PrivateKey: stakePrivateKey,
	}

	staker := Staker{
		Enable:            true,
		Strategy:          "MakeNodes",
		ParentChainWallet: stakerWallet,
	}

	caching := Caching{
		Archive: true,
	}

	restAggregator := RestAggregator{
		Enable: anyTrustMode,
		Urls:   "http://localhost:9876",
	}

	rPCAggregator := RPCAggregator{
		Enable:        anyTrustMode,
		AssumedHonest: 1,
		Backends:      "[{\"url\":\"http://localhost:9876\",\"pubkey\":\"YAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==\",\"signermask\":1}]",
	}

	dataAvailability := DataAvailability{
		Enable:                anyTrustMode,
		SequencerInboxAddress: deployedAddresses.SequencerInbox.Hex(),
		ParentChainNodeURL:    l1conn,
		RestAggregator:        restAggregator,
		RPCAggregator:         rPCAggregator,
	}

	node := Node{
		ForwardingTarget: "",
		Sequencer:        sequencer,
		DelayedSequencer: delayedSequencer,
		BatchPoster:      batchPoster,
		Staker:           staker,
		Caching:          caching,
		DataAvailability: dataAvailability,
	}

	nodeConfig := NodeConfig{
		Chain:       chain,
		ParentChain: parentChain,
		HTTP:        http,
		Node:        node,
	}

	nodeConfigJson, _ := json.Marshal(nodeConfig)
	if err := os.WriteFile(*nodeConfigInfo, nodeConfigJson, 0600); err != nil {
		panic(err)
	}
}
