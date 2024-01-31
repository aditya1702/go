package actions

import (
	"encoding/hex"
	"github.com/stellar/go/network"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"mime"
	"net/http"
)

type envelopeInfo struct {
	hash      string
	innerHash string
	raw       string
	parsed    xdr.TransactionEnvelope
}

func extractEnvelopeInfo(raw string, passphrase string) (envelopeInfo, error) {
	result := envelopeInfo{raw: raw}
	err := xdr.SafeUnmarshalBase64(raw, &result.parsed)
	if err != nil {
		return result, err
	}

	var hash [32]byte
	hash, err = network.HashTransactionInEnvelope(result.parsed, passphrase)
	if err != nil {
		return result, err
	}
	result.hash = hex.EncodeToString(hash[:])
	if result.parsed.IsFeeBump() {
		hash, err = network.HashTransaction(result.parsed.FeeBump.Tx.InnerTx.V1.Tx, passphrase)
		if err != nil {
			return result, err
		}
		result.innerHash = hex.EncodeToString(hash[:])
	}
	return result, nil
}

func validateBodyType(r *http.Request) error {
	c := r.Header.Get("Content-Type")
	if c == "" {
		return nil
	}

	mt, _, err := mime.ParseMediaType(c)
	if err != nil {
		return errors.Wrap(err, "Could not determine mime type")
	}

	if mt != "application/x-www-form-urlencoded" && mt != "multipart/form-data" {
		return &hProblem.UnsupportedMediaType
	}
	return nil
}
