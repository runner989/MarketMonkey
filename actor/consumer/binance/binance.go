package binance

import (
	"errors"
	"fmt"
	"log"
	"marketmonkey/actor/symbol"
	"marketmonkey/event"
	"net"
	"strconv"
	"strings"

	"github.com/anthdm/hollywood/actor"
	"github.com/gorilla/websocket"
	"github.com/valyala/fastjson"
)

const wsEndpoint = "wss://dstream.binance.com/stream?streams="

var symbols = []string{
	"BTCUSDT",
	// "SOLUSDT",
}

type Binance struct {
	ws      *websocket.Conn
	symbols map[string]*actor.PID
	c       *actor.Context
}

func (b *Binance) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.Started:
		b.start(c)
		b.c = c
	}
}

func New() actor.Producer {
	return func() actor.Receiver {
		return &Binance{
			symbols: make(map[string]*actor.PID),
		}
	}
}

func (b *Binance) start(c *actor.Context) {
	// Initialize all the symbol actors as childs
	for _, sym := range symbols {
		pair := event.Pair{
			Exchange: "binance",
			Symbol:   strings.ToLower(sym),
		}
		pid := c.SpawnChild(symbol.New(pair), "symbol", actor.WithID(pair.Symbol))
		b.symbols[pair.Symbol] = pid
	}
	foo := "wss://dstream.binance.com/stream?streams=bnbusdt@aggTrade/btcusdt@markPrice"
	ws, _, err := websocket.DefaultDialer.Dial(foo, nil)
	if err != nil {
		log.Fatal(err)
	}
	b.ws = ws
	go b.wsLoop()
}

func (b *Binance) wsLoop() {
	for {
		_, msg, err := b.ws.ReadMessage()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			fmt.Println("error reading from ws connection", err)
			continue
		}

		parser := fastjson.Parser{}
		v, err := parser.ParseBytes(msg)
		if err != nil {
			fmt.Println("failed to parse msg", err)
			continue
		}
		fmt.Println(v)

		data := v.Get("data")
		stream := string(v.GetStringBytes("stream"))
		symbol, kind := splitStream(stream)

		switch {
		case strings.HasSuffix(stream, "markPrice"):
			b.handleMarkPrice(symbol, data)
		case strings.HasSuffix(stream, "depth"):
			b.handleOrderbook(data)
		}

		if kind == "aggTrade" {
			b.handleAggTrade(symbol, data)
		}
	}
}

func (b *Binance) handleOrderbook(data *fastjson.Value) {
	var (
		asks   = data.GetArray("a")
		bids   = data.GetArray("b")
		symbol = strings.ToLower(string(data.GetStringBytes("s")))
		msg    = event.BookUpdate{
			Unix: data.GetInt64("T"),
			Pair: event.Pair{
				Exchange: "binance",
				Symbol:   symbol,
			},
			Bids: make([]event.BookEntry, 0, len(bids)),
			Asks: make([]event.BookEntry, 0, len(asks)),
		}
	)
	for _, item := range asks {
		price, _ := strconv.ParseFloat(string(item.GetStringBytes("0")), 64)
		size, _ := strconv.ParseFloat(string(item.GetStringBytes("1")), 64)
		msg.Asks = append(msg.Asks, event.BookEntry{
			Price: price,
			Size:  size,
		})
	}
	for _, item := range bids {
		price, _ := strconv.ParseFloat(string(item.GetStringBytes("0")), 64)
		size, _ := strconv.ParseFloat(string(item.GetStringBytes("1")), 64)
		msg.Bids = append(msg.Bids, event.BookEntry{
			Price: price,
			Size:  size,
		})
	}
	b.c.Send(b.symbols[symbol], msg)
}

func (b *Binance) handleMarkPrice(symbol string, data *fastjson.Value) {
	// var (
	// 	unix         = data.GetInt64("E")
	// 	markPriceStr = string(data.GetStringBytes("p"))
	// 	fundingStr   = string(data.GetStringBytes("r"))
	// )
	// funding, _ := strconv.ParseFloat(fundingStr, 64)
	// markPrice, _ := strconv.ParseFloat(markPriceStr, 64)

	// stat := event.Stat{
	// 	Pair: event.Pair{
	// 		Exchange: "binancef",
	// 		Symbol:   symbol,
	// 	},
	// 	MarkPrice: markPrice,
	// 	Funding:   funding,
	// 	Unix:      unix,
	// }
	// symbolPID, _ := b.symbols[symbol]
	// b.c.Send(symbolPID, stat)
	// b.c.Send(b.appPID, stat)
}

func (b *Binance) handleAggTrade(symbol string, data *fastjson.Value) {
	price, _ := strconv.ParseFloat(string(data.GetStringBytes("p")), 64)
	qty, _ := strconv.ParseFloat(string(data.GetStringBytes("q")), 64)
	trade := event.Trade{
		Price: price,
		Qty:   qty,
		IsBuy: data.GetBool("m"),
		Unix:  data.GetInt64("T"),
		Pair: event.Pair{
			Exchange: "binance",
			Symbol:   symbol,
		},
	}
	b.c.Send(b.symbols[symbol], trade)
}

func createWsEndpoint() string {
	results := []string{}
	for _, sym := range symbols {
		results = append(results, fmt.Sprintf("%s@aggTrade", strings.ToLower(sym)))
		results = append(results, fmt.Sprintf("%s@markPrice", strings.ToLower(sym)))
		results = append(results, fmt.Sprintf("%s@depth", strings.ToLower(sym)))
	}
	return fmt.Sprintf("%s%s", wsEndpoint, strings.Join(results, "/"))
}

func splitStream(stream string) (string, string) {
	parts := strings.Split(stream, "@")
	return parts[0], parts[1]
}
