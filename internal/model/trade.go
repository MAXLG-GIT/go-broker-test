package model

import "regexp"

type TradeInput struct {
	Account string  `json:"account"`
	Symbol  string  `json:"symbol"`
	Volume  float64 `json:"volume"`
	Open    float64 `json:"open"`
	Close   float64 `json:"close"`
	Side    string  `json:"side"`
}

type Trade struct {
	ID      int     `db:"id"`
	Account string  `db:"account"`
	Symbol  string  `db:"symbol"`
	Volume  float64 `db:"volume"`
	Open    float64 `db:"open"`
	Close   float64 `db:"close"`
	Side    string  `db:"side"`
}

func (t TradeInput) Validate() bool {
	if t.Account == "" || t.Volume <= 0 || t.Open <= 0 || t.Close <= 0 {
		return false
	}
	// Symbol must be exactly 6 uppercase letters
	match, _ := regexp.MatchString("^[A-Z]{6}$", t.Symbol)
	if !match {
		return false
	}
	if t.Side != "buy" && t.Side != "sell" {
		return false
	}
	return true
}
