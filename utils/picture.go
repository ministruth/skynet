package utils

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/chai2010/webp"
)

// ConvertPicture convert PNG/JPG/WEBP to webp
func ConvertPicture(pic []byte) ([]byte, error) {
	p, _, err := image.Decode(bytes.NewReader(pic))
	if err != nil {
		return nil, err
	}
	b, err := webp.EncodeLosslessRGBA(p)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// ConvertPicture convert PNG/JPG/WEBP to webp base64 string
func ConvertPictureBase64(pic []byte) (string, error) {
	b, err := ConvertPicture(pic)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
