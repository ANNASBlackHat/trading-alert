package model

import "time"

type Kline struct {
	OpenTime int64
	Open     float64
	High     float64
	Low      float64
	Close    float64
}

type ActiveSignal struct {
	EntryPrice float64
	TPPrice    float64 // Red Magic Line (Take-Profit zone)
	SLPrice    float64
	Notified50 bool
	Notified70 bool // "30% closer"
	Notified90 bool // "10% closer"
	SignalTime time.Time
}

type TradeSignal struct {
	Trigger          bool
	EntryPrice       float64
	SLPrice          float64
	TPPrice          float64
	LowestDuringDrop float64
}
