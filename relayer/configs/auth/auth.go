package auth

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"

	. "github.com/henriquemarlon/city.fun/relayer/configs"
	"github.com/henriquemarlon/city.fun/relayer/pkg/ethutil"
)

func GetTransactOpts(chainId *big.Int) (*bind.TransactOpts, error) {
	authKind, err := GetAuthKind()
	if err != nil {
		return nil, err
	}
	switch authKind {
	case AuthKindMnemonicVar, AuthKindMnemonicFile:
		mnemonic, err := GetAuthMnemonic()
		if err != nil {
			return nil, err
		}
		accountIndex, err := GetAuthMnemonicAccountIndex()
		if err != nil {
			return nil, err
		}
		privateKey, err := ethutil.MnemonicToPrivateKey(mnemonic.Value, accountIndex.Value)
		if err != nil {
			return nil, err
		}
		return bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	case AuthKindPrivateKeyVar, AuthKindPrivateKeyFile:
		privateKey, err := GetAuthPrivateKey()
		if err != nil {
			return nil, err
		}
		key, err := crypto.HexToECDSA(ethutil.TrimHex(privateKey.Value))
		if err != nil {
			return nil, err
		}
		return bind.NewKeyedTransactorWithChainID(key, chainId)

	default:
		return nil, fmt.Errorf("no valid authentication method found")
	}
}
