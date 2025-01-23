package event

import "fmt"

type Trade struct {
	Pair  Pair
	Price float64
	Qty   float64
	IsBuy bool
	Unix  int64
}

func (t Trade) GetTimeframe() int64 { return 0 }

type Pair struct {
	Exchange string
	Symbol   string
}

func (p Pair) String() string {
	return fmt.Sprintf("%s %s", p.Exchange, p.Symbol)
}

func NewPair(exchange, symbol string) Pair {
	return Pair{
		Exchange: exchange,
		Symbol:   symbol,
	}
}

type Stat struct {
	Pair      Pair
	MarkPrice float64
	Funding   float64
	Unix      int64
}

type HeatmapLevel struct {
	Price     float64
	Size      float64
	Intensity float64
}

type Heatmap struct {
	PriceGroup float64
	Pair       Pair
	Unix       int64
	Levels     []HeatmapLevel
}

func (h Heatmap) GetTimeframe() int64 { return 0 }

type Candle struct {
	Pair      Pair
	Timeframe int64
	Unix      int64
	Open      float64
	Close     float64
	High      float64
	Low       float64
	Vbuy      float64
	Vsell     float64
	Tbuy      float64
	Tsell     float64
}

func (c Candle) GetTimeframe() int64 { return c.Timeframe }

type Orderbook struct {
	Unix      int64
	Pair      Pair
	AskPrices []float64
	AskSizes  []float64
	AskSums   []float64
	BidPrices []float64
	BidSizes  []float64
	BidSums   []float64
	LastPrice float64
}

func (o Orderbook) GetTimeframe() int64 { return 0 }

type BookUpdate struct {
	Unix int64
	Pair Pair
	Asks []BookEntry
	Bids []BookEntry
}

type BookEntry struct {
	Price float64
	Size  float64
}

type Tick struct {
}

type TickHeatmap struct {
}

type Stream int64

const (
	StreamTrades Stream = iota
	StreamOrderbook
	StreamHeatmap
	StreamCandles
)

type PubSub struct {
	Streams []uint32
}

type PubUnsub struct {
	Streams []uint32
}

type TimeFramer interface {
	GetTimeframe() int64
}
