package blockchain

type TxOutput struct {
	Value  int
	PubKey string
}

type TxInput struct {
	ID  []byte
	Out int
	Sig string
}

func (txi *TxInput) CanUnlock(signature string) bool {
	return txi.Sig == signature
}

func (txi *TxOutput) CanUnlock(signature string) bool {
	return txi.PubKey == signature
}
