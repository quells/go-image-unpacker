/*
Package imageunpacker unpacks a PNG image from a raw binary file.

Expected input file format:
uint16 width, uint16 height,
float32 r, float32 g, float32 b,
...

r, g, b expected to be in range [0, 1]

Throws an error if width or height is greater than 8192 because, yikes,
that's a big image and is probably an error.
*/
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path"
)

// MaxImageDimension is the largest width or height supported for decoded images.
// This is not a technical limitation, just a sort of error-checking in case
// something went wrong with the initial encoding.
const MaxImageDimension = 8192 // pixels

const headerSize = 4    // bytes
const floatSize = 4     // bytes
const elementSize = 12  // bytes
const bufferSize = 1024 // bytes

func main() {
	var input string
	flag.StringVar(&input, "i", "", "Input data filepath")
	var output string
	flag.StringVar(&output, "o", "", "Output image filepath (must be .png)")
	var gamma float64
	flag.Float64Var(&gamma, "gamma", 2.0, "Gamma correction exponent")
	flag.Parse()

	if input == "" || output == "" {
		flag.Usage()
		os.Exit(1)
	}
	if path.Ext(output) != ".png" {
		flag.Usage()
		os.Exit(1)
	}

	bytes, err := getBytes(input)
	if err != nil {
		log.Fatal(err)
	}
	if len(bytes) < headerSize {
		log.Fatalln("File is too small")
	}

	im, err := Unpack(bytes, gamma)
	if err != nil {
		log.Fatal(err)
	}

	err = saveImage(im, output)
	if err != nil {
		log.Fatal(err)
	}
}

// Unpack converts a byte slice to an image and applies a basic gamma correction.
func Unpack(bytes []byte, gamma float64) (*image.RGBA, error) {
	width, height := getDimensions(bytes)
	expectedSize := int(elementSize*width*height) + headerSize
	if len(bytes) != expectedSize {
		return nil, fmt.Errorf("File is corrupted, image size header incorrect")
	}
	if width > MaxImageDimension || height > MaxImageDimension {
		return nil, fmt.Errorf("File is too large, width: %d height: %d", width, height)
	}

	floats := convertToFloat(bytes[headerSize:])
	if gamma != 1.0 {
		floats = gammaCorrect(floats, gamma)
	}

	imgData := standardDynamicRange(floats)
	return makeImage(imgData, width, height), nil
}

func getBytes(filename string) ([]byte, error) {
	reader, err := os.Open(filename)
	defer reader.Close()
	if err != nil {
		return nil, err
	}

	bytes := []byte{}
	buffer := make([]byte, bufferSize)
	for {
		n, err := reader.Read(buffer)
		if err != nil && err.Error() != "EOF" {
			return nil, err
		}
		bytes = append(bytes, buffer[:n]...)
		if n < bufferSize {
			break
		}
	}

	return bytes, nil
}

func getDimensions(bytes []byte) (width, height int) {
	for i := 0; i < headerSize/2; i++ {
		width |= int(bytes[i]) << uint(8*i)
	}
	for i := headerSize / 2; i < headerSize; i++ {
		height |= int(bytes[i]) << uint(8*(i-headerSize/2))
	}
	return
}

func uint32From(bytes []byte) (x uint32) {
	if len(bytes) != 4 {
		log.Fatalln("Expected 4 bytes, got %d: %q", len(bytes), bytes)
	}
	for i := 0; i < 4; i++ {
		x |= uint32(bytes[i]) << uint(8*i)
	}
	return
}

func convertToFloat(bytes []byte) []float64 {
	N := len(bytes) / floatSize
	f := make([]float64, N)
	for i := 0; i < N; i++ {
		ui := uint32From(bytes[floatSize*i : floatSize*(i+1)])
		fi := math.Float32frombits(ui)
		f[i] = float64(fi)
	}
	return f
}

func gammaCorrect(f []float64, g float64) []float64 {
	ig := 1.0 / g
	for i := 0; i < len(f); i++ {
		f[i] = math.Pow(f[i], ig)
	}
	return f
}

func standardDynamicRange(f []float64) []byte {
	bytes := make([]byte, len(f))
	for i := 0; i < len(f); i++ {
		bytes[i] = byte(math.Max(0, math.Min(255, f[i]*255.99)))
	}
	return bytes
}

func makeImage(bytes []byte, w, h int) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	var i int
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			im.Set(x, y, color.NRGBA{bytes[i], bytes[i+1], bytes[i+2], 255})
			i += 3
		}
	}
	return im
}

func saveImage(im *image.RGBA, filename string) error {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		return err
	}
	err = png.Encode(file, im)
	return err
}
