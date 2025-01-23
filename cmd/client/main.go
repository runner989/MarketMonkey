package main

import (
	"log"
	"marketmonkey/actor/consumer/binancef"
	"marketmonkey/app"

	"github.com/anthdm/hollywood/actor"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		log.Fatal(err)
	}

	//engine.Spawn(kraken.New(), "kraken", actor.WithID("1"))
	engine.Spawn(binancef.New(), "binancef", actor.WithID("1"))
	// engine.Spawn(bybit.New(), "bybit", actor.WithID("1"))

	w, h := ebiten.Monitor().Size()
	ebiten.SetWindowSize(w, h)
	ebiten.SetWindowTitle("Market Monkey v.0.01")
	ebiten.SetWindowPosition(0, 0)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	app := app.New(engine)

	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}
