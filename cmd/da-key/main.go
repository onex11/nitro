package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/upgrade_executorgen"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

type Args struct {
	L1Conn          string `json:"l1conn"`
	PrivateKey      string `json:"l1privatekey"`
	SequencerInbox  string `json:"sequencerInbox"`
	UpgradeExecutor string `json:"upgradeExecutor"`
}

func main() {
	setArg, err := ioutil.ReadFile("./rollup-config/set_key.json")
	if err != nil {
		panic(err)
	}
	var arg Args
	json.Unmarshal(setArg, &arg)

	l1client, err := ethclient.Dial(arg.L1Conn)
	if err != nil {
		fmt.Println("dial error: ", err.Error())
		os.Exit(1)
	}

	privateKey, err := crypto.HexToECDSA(arg.PrivateKey)
	if err != nil {
		fmt.Println("get private key error: ", err.Error())
		os.Exit(1)
	}
	pub := privateKey.PublicKey
	fmt.Println(crypto.PubkeyToAddress(pub))
	trOps := bind.NewKeyedTransactor(privateKey)
	gasPrice, err := l1client.SuggestGasPrice(context.Background())
	if err != nil {
		fmt.Println("get suggest gas error: ", err.Error())
		os.Exit(1)
	}
	trOps.GasPrice = gasPrice

	upgradeExecutor, err := upgrade_executorgen.NewUpgradeExecutor(common.HexToAddress(arg.UpgradeExecutor), l1client)
	if err != nil {
		fmt.Println("new upgrade executor: ", err.Error())
		os.Exit(1)
	}

	// _, _, err = das.GenerateAndStoreKeys("./")
	pubkey, err := das.ReadPubKeyFromFile("./rollup-config/das_bls.pub")
	if err != nil {
		fmt.Println("read key error: ", err.Error())
		os.Exit(1)
	}

	keyset := &arbstate.DataAvailabilityKeyset{
		AssumedHonest: 1,
		PubKeys:       []blsSignatures.PublicKey{*pubkey},
	}
	wr := bytes.NewBuffer([]byte{})
	err = keyset.Serialize(wr)
	if err != nil {
		fmt.Println("serialize error: ", err.Error())
		os.Exit(1)
	}
	keysetBytes := wr.Bytes()
	sequencerInboxABI, err := abi.JSON(strings.NewReader(bridgegen.SequencerInboxABI))
	if err != nil {
		fmt.Println("get abi error: ", err.Error())
		os.Exit(1)
	}

	setKeysetCalldata, err := sequencerInboxABI.Pack("setValidKeyset", keysetBytes)
	if err != nil {
		fmt.Println("get call data error: ", err.Error())
		os.Exit(1)
	}
	// dakey, _ := NewDakey(common.HexToAddress(addr), l1client)
	// tx, err := dakey.SetValidKeyset(trOps, setKeysetCalldata)

	tx, err := upgradeExecutor.ExecuteCall(trOps, common.HexToAddress(arg.SequencerInbox), setKeysetCalldata)
	if err != nil {
		fmt.Println("call txn error: ", err.Error())
		os.Exit(1)
	}
	fmt.Printf("txn: %s\n", tx.Hash())

}

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// DakeyMetaData contains all meta data concerning the Dakey contract.
var DakeyMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"keysetBytes\",\"type\":\"bytes\"}],\"name\":\"setValidKeyset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// DakeyABI is the input ABI used to generate the binding from.
// Deprecated: Use DakeyMetaData.ABI instead.
var DakeyABI = DakeyMetaData.ABI

// Dakey is an auto generated Go binding around an Ethereum contract.
type Dakey struct {
	DakeyCaller     // Read-only binding to the contract
	DakeyTransactor // Write-only binding to the contract
	DakeyFilterer   // Log filterer for contract events
}

// DakeyCaller is an auto generated read-only Go binding around an Ethereum contract.
type DakeyCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DakeyTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DakeyTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DakeyFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DakeyFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DakeySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DakeySession struct {
	Contract     *Dakey            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DakeyCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DakeyCallerSession struct {
	Contract *DakeyCaller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// DakeyTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DakeyTransactorSession struct {
	Contract     *DakeyTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DakeyRaw is an auto generated low-level Go binding around an Ethereum contract.
type DakeyRaw struct {
	Contract *Dakey // Generic contract binding to access the raw methods on
}

// DakeyCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DakeyCallerRaw struct {
	Contract *DakeyCaller // Generic read-only contract binding to access the raw methods on
}

// DakeyTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DakeyTransactorRaw struct {
	Contract *DakeyTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDakey creates a new instance of Dakey, bound to a specific deployed contract.
func NewDakey(address common.Address, backend bind.ContractBackend) (*Dakey, error) {
	contract, err := bindDakey(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Dakey{DakeyCaller: DakeyCaller{contract: contract}, DakeyTransactor: DakeyTransactor{contract: contract}, DakeyFilterer: DakeyFilterer{contract: contract}}, nil
}

// NewDakeyCaller creates a new read-only instance of Dakey, bound to a specific deployed contract.
func NewDakeyCaller(address common.Address, caller bind.ContractCaller) (*DakeyCaller, error) {
	contract, err := bindDakey(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DakeyCaller{contract: contract}, nil
}

// NewDakeyTransactor creates a new write-only instance of Dakey, bound to a specific deployed contract.
func NewDakeyTransactor(address common.Address, transactor bind.ContractTransactor) (*DakeyTransactor, error) {
	contract, err := bindDakey(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DakeyTransactor{contract: contract}, nil
}

// NewDakeyFilterer creates a new log filterer instance of Dakey, bound to a specific deployed contract.
func NewDakeyFilterer(address common.Address, filterer bind.ContractFilterer) (*DakeyFilterer, error) {
	contract, err := bindDakey(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DakeyFilterer{contract: contract}, nil
}

// bindDakey binds a generic wrapper to an already deployed contract.
func bindDakey(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := DakeyMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Dakey *DakeyRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Dakey.Contract.DakeyCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Dakey *DakeyRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Dakey.Contract.DakeyTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Dakey *DakeyRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Dakey.Contract.DakeyTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Dakey *DakeyCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Dakey.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Dakey *DakeyTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Dakey.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Dakey *DakeyTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Dakey.Contract.contract.Transact(opts, method, params...)
}

// SetValidKeyset is a paid mutator transaction binding the contract method 0xd1ce8da8.
//
// Solidity: function setValidKeyset(bytes keysetBytes) returns()
func (_Dakey *DakeyTransactor) SetValidKeyset(opts *bind.TransactOpts, keysetBytes []byte) (*types.Transaction, error) {
	return _Dakey.contract.Transact(opts, "setValidKeyset", keysetBytes)
}

// SetValidKeyset is a paid mutator transaction binding the contract method 0xd1ce8da8.
//
// Solidity: function setValidKeyset(bytes keysetBytes) returns()
func (_Dakey *DakeySession) SetValidKeyset(keysetBytes []byte) (*types.Transaction, error) {
	return _Dakey.Contract.SetValidKeyset(&_Dakey.TransactOpts, keysetBytes)
}

// SetValidKeyset is a paid mutator transaction binding the contract method 0xd1ce8da8.
//
// Solidity: function setValidKeyset(bytes keysetBytes) returns()
func (_Dakey *DakeyTransactorSession) SetValidKeyset(keysetBytes []byte) (*types.Transaction, error) {
	return _Dakey.Contract.SetValidKeyset(&_Dakey.TransactOpts, keysetBytes)
}
