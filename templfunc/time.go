package templfunc

import (
	"time"
)

func Time(format string) string {
	return time.Now().Format(format)
}
