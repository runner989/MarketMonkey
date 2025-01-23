package act

import (
	"fmt"
	"marketmonkey/event"

	"github.com/anthdm/hollywood/actor"
)

func GetPublishPID(pair event.Pair) *actor.PID {
	return actor.NewPID("local", fmt.Sprintf("%s/1/symbol/%s/publish/%s", pair.Exchange, pair.Symbol, pair.Symbol))
}
