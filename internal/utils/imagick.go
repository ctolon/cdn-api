package utils

import (
	"log"

	"gopkg.in/gographics/imagick.v2/imagick"
)

// https://github.com/gographics/imagick/tree/master/examples
// https://pkg.go.dev/gopkg.in/gographics/imagick.v3/imagick

func ImagickGetWidthHeight(image []byte) (error, uint, uint) {
	imagick.Initialize()
	defer imagick.Terminate()

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	if err := mw.ReadImageBlob(image); err != nil {
		return err, 0, 0
	}
	return nil, mw.GetImageWidth(), mw.GetImageHeight()
}

func ImagickFormat(image []byte) (error, string) {
	imagick.Initialize()
	defer imagick.Terminate()

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	if err := mw.ReadImageBlob(image); err != nil {
		return err, ""
	}

	return nil, mw.GetImageFormat()
}

func ImagickResize(image []byte, targetWidth, targetHeight uint) []byte {
	imagick.Initialize()
	defer imagick.Terminate()

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	var err error

	err = mw.ReadImageBlob(image)
	if err != nil {
		log.Println("Error reading image:", err)
		return image
	}

	width := mw.GetImageWidth()
	height := mw.GetImageHeight()

	targetWidth, targetHeight = RatioWidthHeight(width, height, targetWidth, targetHeight)

	// Resize the image using the Lanczos filter
	// The blur factor is a float, where > 1 is blurry, < 1 is sharp
	err = mw.ResizeImage(targetWidth, targetHeight, imagick.FILTER_LANCZOS, 1)
	if err != nil {
		log.Println("Error resizing image:", err)
		return image
	}

	// Set the compression quality to 95 (high quality = low compression)
	err = mw.SetImageCompressionQuality(95)
	if err != nil {
		log.Println("Error setting compression quality:", err)
		return image
	}

	// Return byte image
	res, _ := mw.GetImageBlob()
	return res

}
