package consts

import (
	"crypto/sha256"

	"github.com/celestiaorg/nmt/namespace"
)

const (
	// TxInclusionProofQueryPath is the path used to query the application for a
	// tx inclusion proof via the ABCI Query method. The desired transaction
	// index must be formatted into the path.
	TxInclusionProofQueryPath = "custom/txInclusionProof/%d"

	// ShareInclusionProofQueryPath is the path used to query the application for the
	// shares to data root inclusion proofs via the ABCI query method.
	ShareInclusionProofQueryPath = "custom/shareInclusionProof/%d/%d"

	// ProtoBlobTxTypeID is included in each encoded BlobTx to help prevent
	// decoding binaries that are not actually BlobTxs.
	ProtoBlobTxTypeID = "BLOB"

	// ProtoIndexWrapperTypeID is included in each encoded IndexWrapper to help prevent
	// decoding binaries that are not actually IndexWrappers.
	ProtoIndexWrapperTypeID = "INDX"

	// NamespaveVersionSize is the size of a namespace version in bytes.
	NamespaceVersionSize = 1

	// NamespaceIDSize is the size of a namespace ID in bytes.
	NamespaceIDSize = 32

	// NamespaceSize is the size of a namespace in bytes.
	NamespaceSize = NamespaceIDSize + NamespaceVersionSize
)

var (
	// TxNamespaceID is the namespace reserved for transaction data.
	TxNamespaceID = namespace.ID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	// NewBaseHashFunc change accordingly if another hash.Hash should be used as a base hasher in the NMT:
	NewBaseHashFunc = sha256.New

	// DataCommitmentBlocksLimit is the limit to the number of blocks we can generate a data commitment for.
	DataCommitmentBlocksLimit = 1000
)
