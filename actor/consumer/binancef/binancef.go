package binancef

import (
	"errors"
	"fmt"
	"log"
	"marketmonkey/actor/symbol"
	"marketmonkey/event"
	"marketmonkey/settings"
	"net"
	"strconv"
	"strings"

	"github.com/anthdm/hollywood/actor"
	"github.com/gorilla/websocket"
	"github.com/valyala/fastjson"
)

const wsEndpoint = "wss://fstream.binance.com/stream?streams="

type Binancef struct {
	ws      *websocket.Conn
	symbols map[string]*actor.PID
	c       *actor.Context
}

func (b *Binancef) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.Started:
		b.start(c)
		b.c = c
	}
}

func New() actor.Producer {
	return func() actor.Receiver {
		return &Binancef{
			symbols: make(map[string]*actor.PID),
		}
	}
}

func (b *Binancef) start(c *actor.Context) {
	// Initialize all the symbol actors as childs
	market := settings.Markets[settings.Binancef]
	for _, sym := range market.Symbols {
		pair := event.Pair{
			Exchange: "binancef",
			Symbol:   sym.Name,
		}
		pid := c.SpawnChild(symbol.New(pair), "symbol", actor.WithID(pair.Symbol))
		b.symbols[pair.Symbol] = pid
	}
	ws, _, err := websocket.DefaultDialer.Dial(createWsEndpoint(), nil)
	if err != nil {
		log.Fatal(err)
	}
	b.ws = ws
	go b.wsLoop()
}

func (b *Binancef) wsLoop() {
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

func (b *Binancef) handleOrderbook(data *fastjson.Value) {
	var (
		asks   = data.GetArray("a")
		bids   = data.GetArray("b")
		symbol = strings.ToLower(string(data.GetStringBytes("s")))
		msg    = event.BookUpdate{
			Unix: data.GetInt64("T"),
			Pair: event.Pair{
				Exchange: "binancef",
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

func (b *Binancef) handleMarkPrice(symbol string, data *fastjson.Value) {
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

func (b *Binancef) handleAggTrade(symbol string, data *fastjson.Value) {
	price, _ := strconv.ParseFloat(string(data.GetStringBytes("p")), 64)
	qty, _ := strconv.ParseFloat(string(data.GetStringBytes("q")), 64)
	trade := event.Trade{
		Price: price,
		Qty:   qty,
		IsBuy: !data.GetBool("m"),
		Unix:  data.GetInt64("T"),
		Pair: event.Pair{
			Exchange: "binancef",
			Symbol:   symbol,
		},
	}
	b.c.Send(b.symbols[symbol], trade)
}

func createWsEndpoint() string {
	results := []string{}
	for _, sym := range settings.Markets[settings.Binancef].Symbols {
		results = append(results, fmt.Sprintf("%s@aggTrade", strings.ToLower(sym.Name)))
		results = append(results, fmt.Sprintf("%s@markPrice", strings.ToLower(sym.Name)))
		results = append(results, fmt.Sprintf("%s@depth", strings.ToLower(sym.Name)))
	}
	return fmt.Sprintf("%s%s", wsEndpoint, strings.Join(results, "/"))
}

func splitStream(stream string) (string, string) {
	parts := strings.Split(stream, "@")
	return parts[0], parts[1]
}