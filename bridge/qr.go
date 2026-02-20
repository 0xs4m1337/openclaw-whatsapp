package bridge

import (
	"github.com/skip2/go-qrcode"
)

// GenerateQRPNG generates a PNG image of a QR code from the given text.
// Returns PNG bytes. Uses go-qrcode library.
func GenerateQRPNG(qrText string, size int) ([]byte, error) {
	return qrcode.Encode(qrText, qrcode.Medium, size)
}
