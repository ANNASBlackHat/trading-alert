package indicator

import "github.com/annasblackhat/trading-alert/internal/model"

type Indicator interface {
	Analyze(htfKlines []model.Kline, ltfKlines []model.Kline) model.TradeSignal
}
