package hash

import (
	"fmt"
	"image"
	"net/http"

	_ "image/jpeg"
	_ "image/png"

	"gocv.io/x/gocv"
)

const DCT_BLOCK_SIZE = 8

func Phash(img *gocv.Mat) string {
	// resize image to 32x32
	newSize := image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: 32, Y: 32},
	}
	gocv.Resize(*img, img, newSize.Size(), 0, 0, gocv.InterpolationLinear)

	// convert to grayscale
	gocv.CvtColor(*img, img, gocv.ColorBGRToGray)

	// use float32 to represent data
	// https://stackoverflow.com/questions/22117267/how-to-convert-an-image-to-a-float-image-in-opencv
	img.ConvertToWithParams(img, gocv.MatTypeCV32F, 1.0, 0.0)

	gocv.DCT(*img, img, gocv.DftForward)

	// dct_block = dct_block[:8, :8]
	newSize.Max.X = DCT_BLOCK_SIZE
	newSize.Max.Y = DCT_BLOCK_SIZE
	dctBlock := img.Region(newSize)
	dctAverage := float32(dctBlock.Mean().Val1)*float32(64) - dctBlock.GetFloatAt(0, 0)
	dctAverage = dctAverage / float32(63)

	// filter dctblock element < average to 0, else to 1 then add to output
	var out string
	for i := 0; i < DCT_BLOCK_SIZE; i++ {
		for j := 0; j < DCT_BLOCK_SIZE; j++ {
			if dctBlock.GetFloatAt(i, j) < dctAverage {
				// dctBlock.SetFloatAt(i, j, float32(0))
				out += "0"
			} else {
				// dctBlock.SetFloatAt(i, j, float32(1))
				out += "1"
			}
		}
	}

	return out
}

func Dhash(img *gocv.Mat) string {
	// resize image to 8x8
	newSize := image.Rectangle{
		Min: image.Point{0, 0},
		Max: image.Point{8, 8},
	}
	gocv.Resize(*img, img, newSize.Size(), 0, 0, gocv.InterpolationLinear)

	// convert to grayscale
	gocv.CvtColor(*img, img, gocv.ColorBGRToGray)

	var out string
	// TODO: rewrite this!!!!
	for i := 0; i < 8; i++ {
		if i == 0 || i == 7 {
			for j := 1; j < 6; j++ {
				if img.GetIntAt(i, j) < img.GetIntAt(i, j+1) {
					out += "1"
				} else {
					out += "0"
				}
			}
		} else {
			for j := 0; j < 7; j++ {
				if img.GetIntAt(i, j) < img.GetIntAt(i, j+1) {
					out += "1"
				} else {
					out += "0"
				}
			}
		}
	}

	return out
}

func ReadImageFromURL(url string) (*gocv.Mat, error) {
	resp, err := http.Get(url)
	if err != nil || resp.Status != "200 OK" {
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Println(resp.Status)

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, err
	}

	mat, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return nil, err
	}

	return &mat, nil
}
