package delta

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/airbusgeo/godal"
	"github.com/forest-guardian/forest-guardian-api-poc/internal/sentinel"
	"golang.org/x/sync/errgroup"

	"github.com/schollz/progressbar/v3"
)

type Indexes struct {
	NDMI  []float64
	Cloud []float64
	SCL   []float64
	NDRE  []float64
	PSRI  []float64
	B02   []float64
	B04   []float64
	NDVI  []float64
}
type PixelData struct {
	Date time.Time
	X    int
	Y    int
	NDRE float64
	NDMI float64
	PSRI float64
	NDVI float64
}

func createPixelDataset(images map[time.Time]*godal.Dataset) ([]PixelData, error) {
	var width, height, totalPixels int

	for _, imageData := range images {
		width = imageData.Structure().SizeX
		height = imageData.Structure().SizeY
		totalPixels = width * height
		break
	}

	mu := &sync.Mutex{}
	fileResults := []PixelData{}
	count := 0
	target := height * width * len(images)
	progressBar := progressbar.Default(int64(target), "Creating pixel dataset")
	sortedImageDates := getSortedKeys(images)
	eg, _ := errgroup.WithContext(context.Background())

	for y := range height {
		for x := range width {
			eg.Go(func() error {
				for _, date := range sortedImageDates {
					image := images[date]
					result, err := getData(image, totalPixels, width, height, x, y, date)
					if err != nil {
						return err
					}
					count++
					if result != nil {
						mu.Lock()
						fileResults = append(fileResults, *result)
						mu.Unlock()
					}
					if err := progressBar.Add(1); err != nil {
						return err
					}
				}
				return nil
			})
		}
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	if len(fileResults) == 0 {
		return nil, errors.New("no data available to create the dataset")
	}
	return fileResults, nil
}

func getData(image *godal.Dataset, totalPixels, width, height, x, y int, date time.Time) (*PixelData, error) {
	if totalPixels != 0 && totalPixels != width*height {
		return nil, errors.New("different image size")
	}

	indexes, err := sentinel.GetIndexesFromImage(image)
	if err != nil {
		return nil, err
	}

	bands := sentinel.GetBands(indexes, x, y)

	if bands.Valid() {
		return &PixelData{
			Date: date,
			X:    x,
			Y:    y,
			NDRE: bands.NDRE,
			NDMI: bands.NDMI,
			PSRI: bands.PSRI,
			NDVI: bands.NDVI,
		}, nil
	}
	return nil, nil
}

func getSortedKeys(m map[time.Time]*godal.Dataset) []time.Time {
	keys := make([]time.Time, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Before(keys[j])
	})
	return keys
}
