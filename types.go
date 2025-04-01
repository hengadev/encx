package encx

import (
	"time"
)

type intHandler func([]byte, string, int) (string, error)
type stringHandler func([]byte, string, string) (string, error)
type timeHandler func([]byte, string, time.Time) (string, error)
