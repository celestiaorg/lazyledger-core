package core

import (
	abci "github.com/tendermint/tendermint/abci/types"
	cm "github.com/tendermint/tendermint/consensus"
	cmtmath "github.com/tendermint/tendermint/libs/math"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
	"github.com/tendermint/tendermint/types"
)

// Validators gets the validator set at the given block height.
//
// If no height is provided, it will fetch the latest validator set. Note the
// validators are sorted by their voting power - this is the canonical order
// for the validators in the set as used in computing their Merkle root.
//
// More: https://docs.cometbft.com/v0.34/rpc/#/Info/validators
func Validators(ctx *rpctypes.Context, heightPtr *int64, pagePtr, perPagePtr *int) (*ctypes.ResultValidators, error) {
	// The latest validator that we know is the NextValidator of the last block.
	height, err := getHeight(latestUncommittedHeight(), heightPtr)
	if err != nil {
		return nil, err
	}

	validators, err := GetEnvironment().StateStore.LoadValidators(height)
	if err != nil {
		return nil, err
	}

	totalCount := len(validators.Validators)
	perPage := validatePerPage(perPagePtr)
	page, err := validatePage(pagePtr, perPage, totalCount)
	if err != nil {
		return nil, err
	}

	skipCount := validateSkipCount(page, perPage)

	v := validators.Validators[skipCount : skipCount+cmtmath.MinInt(perPage, totalCount-skipCount)]

	return &ctypes.ResultValidators{
		BlockHeight: height,
		Validators:  v,
		Count:       len(v),
		Total:       totalCount}, nil
}

// DumpConsensusState dumps consensus state.
// UNSTABLE
// More: https://docs.cometbft.com/v0.34/rpc/#/Info/dump_consensus_state
func DumpConsensusState(ctx *rpctypes.Context) (*ctypes.ResultDumpConsensusState, error) {
	// Get Peer consensus states.
	peers := GetEnvironment().P2PPeers.Peers().List()
	peerStates := make([]ctypes.PeerStateInfo, len(peers))
	for i, peer := range peers {
		peerState, ok := peer.Get(types.PeerStateKey).(*cm.PeerState)
		if !ok { // peer does not have a state yet
			continue
		}
		peerStateJSON, err := peerState.MarshalJSON()
		if err != nil {
			return nil, err
		}
		peerStates[i] = ctypes.PeerStateInfo{
			// Peer basic info.
			NodeAddress: peer.SocketAddr().String(),
			// Peer consensus state.
			PeerState: peerStateJSON,
		}
	}
	// Get self round state.
	roundState, err := GetEnvironment().ConsensusState.GetRoundStateJSON()
	if err != nil {
		return nil, err
	}
	return &ctypes.ResultDumpConsensusState{
		RoundState: roundState,
		Peers:      peerStates}, nil
}

// ConsensusState returns a concise summary of the consensus state.
// UNSTABLE
// More: https://docs.cometbft.com/v0.34/rpc/#/Info/consensus_state
func ConsensusState(ctx *rpctypes.Context) (*ctypes.ResultConsensusState, error) {
	// Get self round state.
	bz, err := GetEnvironment().ConsensusState.GetRoundStateSimpleJSON()
	return &ctypes.ResultConsensusState{RoundState: bz}, err
}

func ConsensusTimeoutsInfo(ctx *rpctypes.Context,
	heightPtr *int64) (*abci.TimeoutsInfo, error) {
	return GetEnvironment().StateStore.LoadConsensusTimeoutsInfo(*heightPtr)
}

// ConsensusParams gets the consensus parameters at the given block height.
// If no height is provided, it will fetch the latest consensus params.
// More: https://docs.cometbft.com/v0.34/rpc/#/Info/consensus_params
func ConsensusParams(ctx *rpctypes.Context, heightPtr *int64) (*ctypes.ResultConsensusParams, error) {
	// The latest consensus params that we know is the consensus params after the
	// last block.
	height, err := getHeight(latestUncommittedHeight(), heightPtr)
	if err != nil {
		return nil, err
	}

	consensusParams, err := GetEnvironment().StateStore.LoadConsensusParams(height)
	if err != nil {
		return nil, err
	}
	return &ctypes.ResultConsensusParams{
		BlockHeight:     height,
		ConsensusParams: consensusParams}, nil
}
