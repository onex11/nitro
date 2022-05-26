// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
)

type DataAvailabilityServiceWriter interface {
	// Requests that the message be stored until timeout (UTC time in unix epoch seconds).
	Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (*arbstate.DataAvailabilityCertificate, error)
}

type DataAvailabilityService interface {
	arbstate.DataAvailabilityReader
	DataAvailabilityServiceWriter
	fmt.Stringer
}

type DataAvailabilityConfig struct {
	Enable                bool `koanf:"enable"`
	AllowStoreOrigination bool `koanf:"allow-store-origination"`

	LocalCacheConfig BigCacheConfig `koanf:"local-cache"`
	RedisCacheConfig RedisConfig    `koanf:"redis-cache"`

	LocalDBStorageConfig   LocalDBStorageConfig   `koanf:"local-db-storage"`
	LocalFileStorageConfig LocalFileStorageConfig `koanf:"local-file-storage"`
	S3StorageServiceConfig S3StorageServiceConfig `koanf:"s3-storage"`

	KeyConfig KeyConfig `koanf:"key"`

	AggregatorConfig              AggregatorConfig              `koanf:"rpc-aggregator"`
	RestfulClientAggregatorConfig RestfulClientAggregatorConfig `koanf:"rest-aggregator"`

	L1NodeURL             string `koanf:"l1-node-url"`
	SequencerInboxAddress string `koanf:"sequencer-inbox-address"`
}

var DefaultDataAvailabilityConfig = DataAvailabilityConfig{}

/* TODO put these checks somewhere
func (c *DataAvailabilityConfig) Mode() (DataAvailabilityMode, error) {
	if c.ModeImpl == "" {
		return 0, errors.New("--data-availability.mode missing")
	}

	if c.ModeImpl == OnchainDataAvailabilityString {
		return OnchainDataAvailability, nil
	}

	if c.ModeImpl == DASDataAvailabilityString {
		if c.DASConfig.LocalConfig.DataDir == "" || (c.DASConfig.KeyDir == "" && c.DASConfig.PrivKey == "") {
			flag.Usage()
			return 0, errors.New("--data-availability.das.local.data-dir and --data-availability.das.key-dir must be specified if mode is set to das")
		}
		return DASDataAvailability, nil
	}

	if c.ModeImpl == AggregatorDataAvailabilityString {
		if reflect.DeepEqual(c.AggregatorConfig, DefaultAggregatorConfig) {
			flag.Usage()
			return 0, errors.New("--data-availability.aggregator.X config options must be specified if mode is set to aggregator")
		}
		return AggregatorDataAvailability, nil
	}

	flag.Usage()
	return 0, errors.New("--data-availability.mode " + c.ModeImpl + " not recognized")
}
*/

func OptionalAddressFromString(s string) (*common.Address, error) {
	if s == "none" {
		return nil, nil
	}
	if s == "" {
		return nil, errors.New("must provide address for signer or specify 'none'")
	}
	if !common.IsHexAddress(s) {
		return nil, fmt.Errorf("invalid address for signer: %v", s)
	}
	addr := common.HexToAddress(s)
	return &addr, nil
}

func DataAvailabilityConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultDataAvailabilityConfig.Enable, "enable Anytrust Data Availability mode")
	f.Bool(prefix+".allow-store-origination", DefaultDataAvailabilityConfig.AllowStoreOrigination, "allow this server to store new batches; should only be enabled for the batch poster/sequencer")

	// Cache options
	BigCacheConfigAddOptions(prefix+".local-cache", f)
	RedisConfigAddOptions(prefix+".redis-cache", f)

	// Storage options
	LocalDBStorageConfigAddOptions(prefix+".local-db-storage", f)
	LocalFileStorageConfigAddOptions(prefix+".local-file-storage", f)
	S3ConfigAddOptions(prefix+".s3-storage", f)

	// Key config for storage
	KeyConfigAddOptions(prefix+".key", f)

	// Aggregator options
	AggregatorConfigAddOptions(prefix+".rpc-aggregator", f)
	RestfulClientAggregatorConfigAddOptions(prefix+".rest-aggregator", f)

	f.String(prefix+".l1-node-url", DefaultDataAvailabilityConfig.L1NodeURL, "URL for L1 node")
	f.String(prefix+".sequencer-inbox-address", DefaultDataAvailabilityConfig.SequencerInboxAddress, "L1 address of SequencerInbox contract")
}

func serializeSignableFields(c *arbstate.DataAvailabilityCertificate) []byte {
	buf := make([]byte, 0, 32+8)
	buf = append(buf, c.DataHash[:]...)

	var intData [8]byte
	binary.BigEndian.PutUint64(intData[:], c.Timeout)
	buf = append(buf, intData[:]...)

	return buf
}

func Serialize(c *arbstate.DataAvailabilityCertificate) []byte {
	buf := make([]byte, 0)

	buf = append(buf, arbstate.DASMessageHeaderFlag)

	buf = append(buf, c.KeysetHash[:]...)

	buf = append(buf, serializeSignableFields(c)...)

	var intData [8]byte
	binary.BigEndian.PutUint64(intData[:], c.SignersMask)
	buf = append(buf, intData[:]...)

	return append(buf, blsSignatures.SignatureToBytes(c.Sig)...)
}
