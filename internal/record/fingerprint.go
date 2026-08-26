package record

import (
	"fmt"
	"strconv"

	"github.com/cespare/xxhash/v2"
)

func Fingerprint(entry Entry) string {
	h := xxhash.New()
	h.Write([]byte(entry.Source))
	h.Write([]byte{0})
	h.Write([]byte(strconv.FormatFloat(entry.Turbidity, 'g', 6, 64)))
	h.Write([]byte{0})
	h.Write([]byte(strconv.FormatFloat(entry.Chlorine, 'g', 6, 64)))
	h.Write([]byte{0})
	h.Write([]byte(entry.Alarm))
	h.Write([]byte{0})
	h.Write([]byte(entry.At.Format("20060102T150405.000000000")))
	return fmt.Sprintf("%016x", h.Sum64())
}
