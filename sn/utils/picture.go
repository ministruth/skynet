package utils

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/chai2010/webp"
	"github.com/ztrue/tracerr"
)

// WebpImage provides operation for .webp image.
type WebpImage struct {
	Data []byte // image data
}

// Parse converts jpeg and png image to webp type.
func (img *WebpImage) Parse(pic []byte) error {
	srcImg, _, err := image.Decode(bytes.NewReader(pic))
	if err != nil {
		return tracerr.Wrap(err)
	}
	img.Data, err = webp.EncodeLosslessRGBA(srcImg)
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

// Base64 returns base64 of webp image.
func (img *WebpImage) Base64() string {
	return base64.StdEncoding.EncodeToString(img.Data)
}

// ConvertWebp converts jpeg and png image to webp image.
func ConvertWebp(pic []byte) (*WebpImage, error) {
	var ret WebpImage
	if err := tracerr.Wrap(ret.Parse(pic)); err != nil {
		return nil, err
	}
	return &ret, nil
}
