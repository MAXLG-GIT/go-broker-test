package model

type Trade struct {
	Id        int
	Account   string  `json:"account" validate:"required,alphanum"`
	Symbol    string  `json:"symbol"  validate:"required,alpha,len=6"`
	Volume    float64 `json:"volume"  validate:"gt=0"`
	Open      float64 `json:"open"    validate:"gt=0"`
	Close     float64 `json:"close"   validate:"gt=0"`
	Side      string  `json:"side"    validate:"oneof=buy sell"`
	Processed int
}

//func (tr *Trade) ProcessTrade() error {
//
//	return nil
//}
