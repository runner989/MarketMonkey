package orderbook

import (
	"marketmonkey/event"
	"math"
	"sort"
)

func (o *Orderbook) calculateHeatmap() event.Heatmap {
	depth := 500
	bidMap := map[float64]float64{}
	maxSize := 0.0
	unix := o.lastUnix / 1000

	o.bids.Descend(1000000, func(price float64, size float64) bool {
		if len(bidMap) == depth {
			return false
		}
		groupedPrice := math.Floor(price/o.priceGroup) * o.priceGroup
		bidMap[groupedPrice] += size
		return true
	})

	askMap := map[float64]float64{}
	o.asks.Ascend(0, func(price float64, size float64) bool {
		if len(askMap) == depth {
			return false
		}
		groupedPrice := math.Floor(price/o.priceGroup) * o.priceGroup
		askMap[groupedPrice] += size
		return true
	})

	for _, size := range bidMap {
		if size > maxSize {
			maxSize = size
		}
	}
	for _, size := range askMap {
		if size > maxSize {
			maxSize = size
		}
	}

	return event.Heatmap{
		PriceGroup: o.priceGroup,
		Unix:       unix,
		Pair:       o.pair,
		Levels:     flattenAndSort(bidMap, askMap, maxSize),
	}
}

func flattenAndSort(bids map[float64]float64, asks map[float64]float64, maxSize float64) []event.HeatmapLevel {
	levels := make([]event.HeatmapLevel, len(bids)+len(asks))

	i := 0
	for price, size := range asks {
		val := math.Log10(size+1) / math.Log10(maxSize+1)
		levels[i] = event.HeatmapLevel{
			Price:     price,
			Size:      size,
			Intensity: clamp(val, 0, 1),
		}
		i++
	}
	for price, size := range bids {
		val := math.Log10(size+1) / math.Log10(maxSize+1)
		levels[i] = event.HeatmapLevel{
			Price:     price,
			Size:      size,
			Intensity: clamp(val, 0, 1),
		}
		i++
	}

	sort.Slice(levels, func(i, j int) bool {
		return levels[i].Price < levels[j].Price
	})

	return levels
}

func clamp(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
