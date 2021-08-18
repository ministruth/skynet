package utils

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/chai2010/webp"
)

// WebpImage provides operation for .webp image.
type WebpImage struct {
	Data []byte // image data
}

// Parse converts jpeg and png image to webp type.
func (img *WebpImage) Parse(pic []byte) error {
	var err error
	srcImg, _, err := image.Decode(bytes.NewReader(pic))
	if err != nil {
		return err
	}
	img.Data, err = webp.EncodeLosslessRGBA(srcImg)
	if err != nil {
		return err
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
	err := ret.Parse(pic)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}
