package state_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbm "github.com/tendermint/tm-db"

	tmrand "github.com/lazyledger/lazyledger-core/libs/rand"
	sm "github.com/lazyledger/lazyledger-core/state"
	"github.com/lazyledger/lazyledger-core/types"
)

func TestTxFilter(t *testing.T) {
	genDoc := randomGenesisDoc()
	genDoc.ConsensusParams.Block.MaxBytes = 3000
	genDoc.ConsensusParams.Evidence.MaxBytes = 1500

	// Max size of Txs is much smaller than size of block,
	// since we need to account for commits and evidence.
	testCases := []struct {
		tx    types.Tx
		isErr bool
	}{
		{types.Tx{Value: tmrand.Bytes(2178)}, false},
		{types.Tx{Value: tmrand.Bytes(2179)}, true},
		{types.Tx{Value: tmrand.Bytes(3000)}, true},
	}

	for i, tc := range testCases {
		stateDB, err := dbm.NewDB("state", "memdb", os.TempDir())
		require.NoError(t, err)
		stateStore := sm.NewStore(stateDB)
		state, err := stateStore.LoadFromDBOrGenesisDoc(genDoc)
		require.NoError(t, err)

		f := sm.TxPreCheck(state)
		if tc.isErr {
			assert.NotNil(t, f(tc.tx), "#%v", i)
		} else {
			assert.Nil(t, f(tc.tx), "#%v", i)
		}
	}
}
