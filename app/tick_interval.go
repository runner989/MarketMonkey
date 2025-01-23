package app

type TickInterval int64

func (t TickInterval) String() string {
	switch t {
	case 1:
		return "1S"
	case 5:
		return "5S"
	case 60:
		return "1M"
	case 300:
		return "5M"
	case 900:
		return "15M"
	case 3600:
		return "1H"
	case 86400:
		return "1D"
	case 604800:
		return "1W"
	case 2629800:
		return "1M"
	default:
		return "1S"
	}
}
