package coinbase

import (
	"errors"
	"fmt"
	"log"
	"marketmonkey/actor/symbol"
	"marketmonkey/event"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/gorilla/websocket"
	"github.com/valyala/fastjson"
)

const wsEndpoint = "wss://ws-feed.exchange.coinbase.com"

var symbols = []string{
	"BTC-USD",
	"ETH-USD",
	"SOL-USD",
}

type Coinbase struct {
	ws      *websocket.Conn
	symbols map[string]*actor.PID
	c       *actor.Context
}

func (b *Coinbase) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.Started:
		b.start(c)
		b.c = c
	}
}

func New() actor.Producer {
	return func() actor.Receiver {
		return &Coinbase{
			symbols: make(map[string]*actor.PID),
		}
	}
}

func (b *Coinbase) start(c *actor.Context) {
	// Initialize all the symbol actors as children
	for _, sym := range symbols {
		pair := event.Pair{
			Exchange: "coinbase",
			Symbol:   strings.ToLower(strings.Replace(sym, "-", "", -1)),
		}
		pid := c.SpawnChild(symbol.New(pair), "symbol", actor.WithID(pair.Symbol))
		b.symbols[pair.Symbol] = pid
	}

	ws, _, err := websocket.DefaultDialer.Dial(wsEndpoint, nil)
	if err != nil {
		log.Fatal(err)
	}
	b.ws = ws

	subscribeMsg := map[string]interface{}{
		"type": "subscribe",
		"channels": []map[string]interface{}{
			{
				"name":        "level2_batch",
				"product_ids": symbols,
			},
			{
				"name":        "matches",
				"product_ids": symbols,
			},
		},
	}
	if err := ws.WriteJSON(subscribeMsg); err != nil {
		log.Fatal(err)
	}

	go b.wsLoop()
}

func (b *Coinbase) wsLoop() {
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

		msgType := string(v.GetStringBytes("type"))
		switch msgType {
		case "snapshot":
			b.handleSnapshot(v)
		case "l2update":
			b.handleOrderbook(v)
		case "match":
			b.handleTrade(v)
		}
	}
}

func (b *Coinbase) handleSnapshot(data *fastjson.Value) {
	productID := string(data.GetStringBytes("product_id"))
	symbol := strings.ToLower(strings.Replace(productID, "-", "", -1))

	bidsValue := data.Get("bids")
	asksValue := data.Get("asks")
	if bidsValue == nil || asksValue == nil {
		return
	}

	bids := bidsValue.GetArray()
	asks := asksValue.GetArray()

	msg := event.BookUpdate{
		Unix: parseTimestamp(string(data.GetStringBytes("time"))),
		Pair: event.Pair{
			Exchange: "coinbase",
			Symbol:   symbol,
		},
		Bids: make([]event.BookEntry, 0, len(bids)),
		Asks: make([]event.BookEntry, 0, len(asks)),
	}

	for _, bid := range bids {
		bidArr := bid.GetArray()
		if len(bidArr) < 2 {
			continue
		}
		price, err := strconv.ParseFloat(string(bidArr[0].GetStringBytes()), 64)
		if err != nil {
			continue
		}
		size, err := strconv.ParseFloat(string(bidArr[1].GetStringBytes()), 64)
		if err != nil {
			continue
		}
		msg.Bids = append(msg.Bids, event.BookEntry{
			Price: price,
			Size:  size,
		})
	}

	for _, ask := range asks {
		askArr := ask.GetArray()
		if len(askArr) < 2 {
			continue
		}
		price, err := strconv.ParseFloat(string(askArr[0].GetStringBytes()), 64)
		if err != nil {
			continue
		}
		size, err := strconv.ParseFloat(string(askArr[1].GetStringBytes()), 64)
		if err != nil {
			continue
		}
		msg.Asks = append(msg.Asks, event.BookEntry{
			Price: price,
			Size:  size,
		})
	}

	b.c.Send(b.symbols[symbol], msg)
}

func (b *Coinbase) handleOrderbook(data *fastjson.Value) {
	productID := string(data.GetStringBytes("product_id"))
	symbol := strings.ToLower(strings.Replace(productID, "-", "", -1))

	changesValue := data.Get("changes")
	if changesValue == nil {
		return
	}
	changes := changesValue.GetArray()

	msg := event.BookUpdate{
		Unix: parseTimestamp(string(data.GetStringBytes("time"))),
		Pair: event.Pair{
			Exchange: "coinbase",
			Symbol:   symbol,
		},
		Bids: make([]event.BookEntry, 0),
		Asks: make([]event.BookEntry, 0),
	}

	for _, change := range changes {
		changeArr := change.GetArray()
		if len(changeArr) != 3 {
			continue
		}

		side := string(changeArr[0].GetStringBytes())
		price, err := strconv.ParseFloat(string(changeArr[1].GetStringBytes()), 64)
		if err != nil {
			continue
		}
		size, err := strconv.ParseFloat(string(changeArr[2].GetStringBytes()), 64)
		if err != nil {
			continue
		}

		entry := event.BookEntry{
			Price: price,
			Size:  size,
		}

		if side == "buy" {
			msg.Bids = append(msg.Bids, entry)
		} else if side == "sell" {
			msg.Asks = append(msg.Asks, entry)
		}
	}

	b.c.Send(b.symbols[symbol], msg)
}

func (b *Coinbase) handleTrade(data *fastjson.Value) {
	productID := string(data.GetStringBytes("product_id"))
	symbol := strings.ToLower(strings.Replace(productID, "-", "", -1))

	price, _ := strconv.ParseFloat(string(data.GetStringBytes("price")), 64)
	size, _ := strconv.ParseFloat(string(data.GetStringBytes("size")), 64)
	side := string(data.GetStringBytes("side"))

	trade := event.Trade{
		Price: price,
		Qty:   size,
		IsBuy: side == "buy",
		Unix:  parseTimestamp(string(data.GetStringBytes("time"))),
		Pair: event.Pair{
			Exchange: "coinbase",
			Symbol:   symbol,
		},
	}

	b.c.Send(b.symbols[symbol], trade)
}

// parseTimestamp converts Coinbase's ISO8601 timestamp to Unix milliseconds
func parseTimestamp(ts string) int64 {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return time.Now().UnixMilli()
	}
	return t.UnixMilli()
}
