package sampling

import (
	"math"
	"time"

	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat"
)

type ConfidenceIntervalGenerator struct {
	rng *rand.Rand
}

func NewConfidenceIntervalGenerator() *ConfidenceIntervalGenerator {
	source := rand.NewSource(uint64(time.Now().UnixNano()))
	return &ConfidenceIntervalGenerator{
		rng: rand.New(source),
	}
}

func (g *ConfidenceIntervalGenerator) GenerateValue(min, max int, factor float64) float64 {
	lowerBound, upperBound := calculateConfidenceInterval(min, max, factor)

	randomRange := upperBound - lowerBound
	value := g.rng.Float64()*randomRange + lowerBound

	return math.Round(value)
}

func calculateConfidenceInterval(min, max int, factor float64) (float64, float64) {
	intervalValues := make([]float64, int(max-min)+1)
	for i := range intervalValues {
		intervalValues[i] = float64(min) + float64(i)
	}

	mean, stdDev := stat.MeanStdDev(intervalValues, nil)
	confidenceFactor := stdDev / math.Sqrt(float64(len(intervalValues)))

	lowerBound := mean - factor*confidenceFactor
	upperBound := mean + factor*confidenceFactor

	return lowerBound, upperBound
}