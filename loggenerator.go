package loggenerator

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

const (
	LvlFatal = iota
	LvlError
	LvlWarn
	LvlInfo
	LvlDebug
	LvlTrace
)

// LevelRatios where used will should be referenced by the Lvl... constants
type LevelRatios []decimal.Decimal

type LogGenerator struct {
	Specified map[int64]bool
	Ratios    LevelRatios
	Ranges    []int64
	Logger    *logrus.Logger
}

// New creates a LogGenerator, automatically adjusting unspecified ratios so that
// the total always equals UpperLimit. It returns any validation errors.
func New(specified map[int64]bool, ratios LevelRatios) (LogGenerator, []error) {
	generator := LogGenerator{
		Logger:    logrus.New(),
		Specified: specified,
		Ratios:    ratios,
	}

	// validation if there were no specified overrides
	if len(specified) == 0 {
		if errs := generator.validateLevelRatios(); len(errs) != 0 {
			return LogGenerator{}, errs
		}
	}
	return generator, nil
}

// DeriveDistributionRanges take the specified percentages, finds the gaps, and
// redistruted the unspecified length evenly amongst the unspecified levels
// with the exception of Fatal, since that should only ever be set by the caller
func (g LogGenerator) DeriveDistributionRanges() []int64 {
	oneHundred := decimal.NewFromInt(100)

	inputsTotal := decimal.NewFromInt(int64(len(g.Ratios)))
	rangeTotalLength := inputsTotal.Mul(oneHundred)

	totalSpecified := int64(len(g.Specified))

	// figure out how much was specified so that we can subtract it
	// to know what is left to redistribute
	pctTotal := decimal.Zero
	for _, pct := range g.Ratios {
		pctTotal = pctTotal.Add(pct)
	}

	remainingPct := oneHundred
	if !pctTotal.Equal(oneHundred) {
		// convert the left over percent to an individual portion
		remainingPct = remainingPct.Sub(pctTotal)
		specifiedCount := decimal.NewFromInt(totalSpecified)
		remainingCount := decimal.NewFromInt(int64(len(g.Ratios))).Sub(specifiedCount)
		individualPortion := remainingPct.Div(remainingCount)

		// calculate Fatal's share and tack it onto the individualPortion
		// when Fatal hasn't been specified
		if !g.Specified[LvlFatal] {
			fatalPortion := individualPortion.Div(remainingCount.Sub(decimal.NewFromInt(1)))
			individualPortion = individualPortion.Add(fatalPortion)
		}

		// redistribute left over to unspecified levels
		for idx := range g.Ratios {
			if !g.Specified[int64(idx)] && idx != 0 {
				g.Ratios[idx] = individualPortion
			}
		}
	}

	// now round up to avoid tiny gaps and turn back to ints
	outputs := make([]int64, len(g.Ratios))
	for idx := range g.Ratios {
		multiplied := g.Ratios[idx].Mul(rangeTotalLength)
		outputs[idx] = multiplied.RoundUp(int32(2)).IntPart()
	}
	return outputs
}

// Output generates logs according to the ratios
func (g LogGenerator) Output(ranges []int64, outputLimit int, delay int64) map[int64]int64 {
	rangeLimit := int64(60000)
	fatalLow := rangeLimit - ranges[LvlFatal]
	errorLow := fatalLow - ranges[LvlError]
	warnLow := errorLow - ranges[LvlWarn]
	infoLow := warnLow - ranges[LvlInfo]
	debugLow := infoLow - ranges[LvlDebug]
	// traceLow := debugLow - ranges[LvlTrace] is unneccesary as it should always be 0

	seed := rand.NewSource(time.Now().UnixNano())
	rander := rand.New(seed)
	outputCounts := make(map[int64]int64, 6)
	for i := 0; i <= outputLimit; i++ {
		time.Sleep(time.Millisecond * time.Duration(delay))
		randOut := rander.Int63n(int64(rangeLimit))
		switch {
		case randOut >= fatalLow && randOut < rangeLimit:
			outputCounts[LvlFatal]++
			g.Logger.Fatalf("fatal level message")
		case randOut >= errorLow && randOut < fatalLow:
			outputCounts[LvlError]++
			g.Logger.Errorf("error level message")
		case randOut >= warnLow && randOut < errorLow:
			outputCounts[LvlWarn]++
			g.Logger.Warnf("warn level message")
		case randOut >= infoLow && randOut < warnLow:
			outputCounts[LvlInfo]++
			g.Logger.Infof("info level message")
		case randOut >= debugLow && randOut < infoLow:
			outputCounts[LvlDebug]++
			g.Logger.Debugf("debug level message")
		case randOut > 0 && randOut < debugLow:
			outputCounts[LvlTrace]++
			g.Logger.Tracef("trace level message")
		}
	}
	return outputCounts
}

// validateLevelRatios ensures each ratio is within [0, UpperLimit] and that the sum
// equals UpperLimit exactly.
func (g LogGenerator) validateLevelRatios() []error {
	var errors []error
	oneHundred := decimal.NewFromInt(100)
	if g.Ratios[LvlFatal].GreaterThan(oneHundred) || g.Ratios[LvlFatal].LessThan(decimal.Zero) {
		errors = append(
			errors,
			fmt.Errorf(
				"fatal level is outside possible range with value %s",
				g.Ratios[LvlFatal].String(),
			),
		)
	}
	if g.Ratios[LvlError].GreaterThan(oneHundred) || g.Ratios[LvlError].LessThan(decimal.Zero) {
		errors = append(
			errors,
			fmt.Errorf(
				"error level is outside possible range with value %s",
				g.Ratios[LvlError].String(),
			),
		)
	}
	if g.Ratios[LvlWarn].GreaterThan(oneHundred) || g.Ratios[LvlWarn].LessThan(decimal.Zero) {
		errors = append(
			errors,
			fmt.Errorf(
				"warn level is outside possible range with value %s",
				g.Ratios[LvlWarn].String(),
			),
		)
	}
	if g.Ratios[LvlInfo].GreaterThan(oneHundred) || g.Ratios[LvlInfo].LessThan(decimal.Zero) {
		errors = append(
			errors,
			fmt.Errorf(
				"info level is outside possible range with value %s",
				g.Ratios[LvlInfo].String(),
			),
		)
	}
	if g.Ratios[LvlDebug].GreaterThan(oneHundred) || g.Ratios[LvlDebug].LessThan(decimal.Zero) {
		errors = append(
			errors,
			fmt.Errorf(
				"debug level is outside possible range with value %s",
				g.Ratios[LvlDebug].String(),
			),
		)
	}
	if g.Ratios[LvlTrace].GreaterThan(oneHundred) || g.Ratios[LvlTrace].LessThan(decimal.Zero) {
		errors = append(
			errors,
			fmt.Errorf(
				"trace level is outside possible range with value %s",
				g.Ratios[LvlTrace].String(),
			),
		)
	}

	sum := g.Ratios[LvlFatal].IntPart() + g.Ratios[LvlError].IntPart() + g.Ratios[LvlWarn].IntPart() + g.Ratios[LvlInfo].IntPart() + g.Ratios[LvlDebug].IntPart() + g.Ratios[LvlTrace].IntPart()
	if sum != 100 {
		errors = append(errors,
			fmt.Errorf("log level ratio sum must equal 100, got %d", sum))
	}
	return errors
}
