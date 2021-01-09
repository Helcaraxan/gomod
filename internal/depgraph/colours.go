package depgraph

import (
	"fmt"
	"math"
	"math/bits"
)

func hashToColourHSV(hash string, isTest bool) (text string, background string) {
	var h byte
	for _, b := range []byte(hash) {
		h ^= bits.RotateLeft8(uint8(b), int(b))
	}
	hue := float32(uint8(h)) / float32(math.MaxUint8)
	satVar := float32(uint8(h^0xff)) / float32(math.MaxUint8)

	text = "0.000 0.000 0.000"
	sat := 0.7 + 0.3*satVar
	if isTest {
		sat = 0.2 + 0.2*satVar
	} else if hue < 0.10 || (hue > 0.6 && hue < 0.8) {
		text = "0.000 0.000 1.000"
	}
	return text, fmt.Sprintf("%.3f %.3f 1.000", hue, sat)
}
