package model

type Account struct {
	AccountId string  `json:"account"`
	Trades    int     `json:"trades"`
	Profit    float64 `json:"profit"`
}
