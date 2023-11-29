# deploy-templates

## Steps to deploy
1. Edit `deploy_config.json`
   - `l1conn` and `l1ChainIdUint` - L2 info (Arbitrum Sepolia or Arbitrum One)
   - `ownerAddressString` and `l1privatekey` - Initial owner account
   - `sequencerAddressString` - BatchPoster account
   - `l2chainname` - L3 network name
   - `prod` - `false` for testnet
2. Edit `l2_chain_config.json`
   - `chainId` - L3 chain ID
   - `arbitrum.DataAvailabilityCommittee` - `true` if launching AnyTrust mode
   - `InitialChainOwner` - Initial owner account (same account from `deploy_config.json`)
3. Execute createRollup (below is run from repo clone root folder)
   ```
   mkdir deploy-result
   mkdir -p deploy-result/inputs
   mkdir -p deploy-result/outputs
   cp deploy-templates/* deploy-result/inputs/
   docker run --rm \
      --volume $(pwd)/deploy-result/inputs:/rollup-config \
      --volume $(pwd)/deploy-result/outputs:/rollup-deployment \
      305587085711.dkr.ecr.us-west-2.amazonaws.com/orbit-createrollup:latest 2>&1 | tee deploy-result/docker-run-log.txt
   ```
4. Keep all files under `deploy-result`