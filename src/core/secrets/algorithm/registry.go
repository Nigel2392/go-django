package algorithm

import (
	"crypto/sha256"
)

var signature_algos = make(map[string]Algorithm)

func init() {
	var name = "sha256"

	RegisterSignatureAlgos(NewSignatureAlgorithm(name, sha256.New))
}

func GetSignatureAlgo(name string) (Algorithm, bool) {
	algo, ok := signature_algos[name]
	return algo, ok
}

func RegisterSignatureAlgos(algo ...Algorithm) {
	for _, a := range algo {
		signature_algos[a.Name()] = a
	}
}
