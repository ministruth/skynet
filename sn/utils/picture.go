package utils

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/chai2010/webp"
)

type WebpImage struct {
	data []byte
}

func (img *WebpImage) Data() []byte {
	return img.data
}

func (img *WebpImage) Parse(pic []byte) error {
	p, _, err := image.Decode(bytes.NewReader(pic))
	if err != nil {
		return err
	}
	b, err := webp.EncodeLosslessRGBA(p)
	if err != nil {
		return err
	}
	img.data = b
	return nil
}

func (img *WebpImage) Base64() string {
	return base64.StdEncoding.EncodeToString(img.data)
}

func PicFromByte(pic []byte) (*WebpImage, error) {
	var ret WebpImage
	err := ret.Parse(pic)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}
