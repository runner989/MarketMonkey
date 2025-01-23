package bybit

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

const wsEndpoint = "wss://stream.bybit.com/v5/public/linear"

var symbols = []string{
	"BTCUSDT",
	"ETHUSDT",
	"SOLUSDT",
}

type Bybit struct {
	ws      *websocket.Conn
	symbols map[string]*actor.PID
	c       *actor.Context
}

func (b *Bybit) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.Started:
		b.start(c)
		b.c = c
	case actor.Stopped:
		if b.ws != nil {
			b.ws.Close()
		}
	}
}

func New() actor.Producer {
	return func() actor.Receiver {
		return &Bybit{
			symbols: make(map[string]*actor.PID),
		}
	}
}

func (b *Bybit) start(c *actor.Context) {
	// Initialize all the symbol actors as children
	for _, sym := range symbols {
		pair := event.Pair{
			Exchange: "bybit",
			Symbol:   strings.ToLower(sym),
		}
		pid := c.SpawnChild(symbol.New(pair), "symbol", actor.WithID(pair.Symbol))
		b.symbols[pair.Symbol] = pid
	}

	ws, _, err := websocket.DefaultDialer.Dial(createWsEndpoint(), nil)
	if err != nil {
		log.Fatal(err)
	}
	b.ws = ws

	if err := b.subscribe(); err != nil {
		log.Fatal(err)
	}

	go b.heartbeat()
	go b.wsLoop()
}

func createWsEndpoint() string {
	return wsEndpoint
}

func (b *Bybit) subscribe() error {
	streams := make([]string, 0, len(symbols)*2)
	for _, sym := range symbols {
		streams = append(streams, fmt.Sprintf("orderbook.50.%s", sym)) // orderbook stream (50 levels - 20ms frequency)
		streams = append(streams, fmt.Sprintf("publicTrade.%s", sym))
	}

	subMsg := map[string]interface{}{
		"req_id": "marketmonkey",
		"op":     "subscribe",
		"args":   streams,
	}

	log.Printf("Subscribing to Bybit streams: %v", streams)
	return b.ws.WriteJSON(subMsg)
}

func (b *Bybit) wsLoop() {
	for {
		_, msg, err := b.ws.ReadMessage()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			log.Printf("error reading from ws connection: %v", err)
			continue
		}

		parser := fastjson.Parser{}
		v, err := parser.ParseBytes(msg)
		if err != nil {
			log.Printf("failed to parse msg: %v", err)
			continue
		}

		if v.Exists("success") {
			success := v.GetBool("success")
			op := string(v.GetStringBytes("op"))
			log.Printf("Received control message: success=%v op=%s", success, op)
			continue
		}

		topic := string(v.GetStringBytes("topic"))

		if strings.HasPrefix(topic, "publicTrade") {
			b.handleTrade(v)
		} else if strings.HasPrefix(topic, "orderbook") {
			b.handleOrderbook(v)
		}
	}
}

func (b *Bybit) handleOrderbook(v *fastjson.Value) {
	data := v.Get("data")
	if data == nil {
		log.Printf("No data field in orderbook message")
		return
	}

	topic := string(v.GetStringBytes("topic"))
	msgType := string(v.GetStringBytes("type"))
	parts := strings.Split(topic, ".")
	if len(parts) < 3 {
		log.Printf("Invalid topic format: %s", topic)
		return
	}
	symbol := strings.ToLower(parts[2])

	bids := data.GetArray("b")
	asks := data.GetArray("a")
	if len(bids) == 0 && len(asks) == 0 {
		return
	}

	msg := event.BookUpdate{
		Unix: v.GetInt64("ts") / 1000,
		Pair: event.Pair{
			Exchange: "bybit",
			Symbol:   symbol,
		},
		Bids: make([]event.BookEntry, 0, 50),
		Asks: make([]event.BookEntry, 0, 50),
	}

	for _, item := range asks {
		arr := item.GetArray()
		if len(arr) < 2 {
			continue
		}
		price, _ := strconv.ParseFloat(string(arr[0].GetStringBytes()), 64)
		size, _ := strconv.ParseFloat(string(arr[1].GetStringBytes()), 64)

		if msgType == "delta" && size == 0 {
			continue
		}

		msg.Asks = append(msg.Asks, event.BookEntry{
			Price: price,
			Size:  size,
		})
	}

	for _, item := range bids {
		arr := item.GetArray()
		if len(arr) < 2 {
			continue
		}
		price, _ := strconv.ParseFloat(string(arr[0].GetStringBytes()), 64)
		size, _ := strconv.ParseFloat(string(arr[1].GetStringBytes()), 64)

		if msgType == "delta" && size == 0 {
			continue
		}

		msg.Bids = append(msg.Bids, event.BookEntry{
			Price: price,
			Size:  size,
		})
	}

	if len(msg.Asks) == 0 && len(msg.Bids) == 0 {
		return
	}

	if pid, ok := b.symbols[symbol]; ok {
		b.c.Send(pid, msg)
	} else {
		log.Printf("No symbol actor found for %s", symbol)
	}
}

func (b *Bybit) handleTrade(v *fastjson.Value) {
	data := v.Get("data")
	if data == nil {
		log.Printf("No data field in trade message")
		return
	}

	topic := string(v.GetStringBytes("topic"))
	parts := strings.Split(topic, ".")
	if len(parts) < 2 {
		log.Printf("Invalid topic format: %s", topic)
		return
	}
	symbol := strings.ToLower(parts[1])

	trades := data.GetArray()
	if len(trades) == 0 {
		return
	}

	for _, t := range trades {
		price, _ := strconv.ParseFloat(string(t.GetStringBytes("p")), 64)
		qty, _ := strconv.ParseFloat(string(t.GetStringBytes("v")), 64)
		side := string(t.GetStringBytes("S"))
		timestamp := t.GetInt64("T")

		trade := event.Trade{
			Price: price,
			Qty:   qty,
			IsBuy: side == "Buy",
			Unix:  timestamp,
			Pair: event.Pair{
				Exchange: "bybit",
				Symbol:   symbol,
			},
		}

		if pid, ok := b.symbols[symbol]; ok {
			b.c.Send(pid, trade)
		} else {
			log.Printf("No symbol actor found for %s", symbol)
		}
	}
}

func (b *Bybit) heartbeat() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		pingMsg := map[string]interface{}{
			"req_id": "ping",
			"op":     "ping",
		}
		if err := b.ws.WriteJSON(pingMsg); err != nil {
			log.Printf("Failed to send ping: %v", err)
			return
		}
	}
}
