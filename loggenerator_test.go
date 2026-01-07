package loggenerator

import (
	"io"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LogGenerator(t *testing.T) {

	tests := []struct {
		name               string
		specified          map[int64]bool
		levelRatios        []decimal.Decimal
		expectedRanges     []int64
		wantError          bool
		expectedCountFatal int64
		expectedCountError int64
		expectedCountWarn  int64
		expectedCountInfo  int64
		expectedCountDebug int64
		expectedCountTrace int64
	}{
		{
			name:      "no defaults, no overrides, triggers validation error",
			wantError: true,
			specified: map[int64]bool{},
			levelRatios: []decimal.Decimal{
				decimal.NewFromInt(0), // Fatal
				decimal.NewFromInt(0), // Error
				decimal.NewFromInt(0), // Warn
				decimal.NewFromInt(0), // Info
				decimal.NewFromInt(0), // Debug
				decimal.NewFromInt(0), // Trace
			},
			expectedRanges:     []int64{},
			expectedCountFatal: int64(0),
			expectedCountError: int64(0),
			expectedCountWarn:  int64(0),
			expectedCountInfo:  int64(0),
			expectedCountDebug: int64(0),
			expectedCountTrace: int64(0),
		},
		{
			name:      "sane defaults, no errors",
			wantError: false,
			specified: map[int64]bool{},
			levelRatios: []decimal.Decimal{
				decimal.NewFromInt(0),  // Fatal
				decimal.NewFromInt(10), // Error
				decimal.NewFromInt(20), // Warn
				decimal.NewFromInt(50), // Info
				decimal.NewFromInt(20), // Debug
				decimal.NewFromInt(0),  // Trace
			},
			expectedRanges:     []int64{0, 6000, 12000, 30000, 12000, 0},
			expectedCountFatal: int64(0),
			expectedCountError: int64(200),
			expectedCountWarn:  int64(400),
			expectedCountInfo:  int64(1000),
			expectedCountDebug: int64(400),
			expectedCountTrace: int64(0),
		},
		{
			name:      "override, 100 pct info level, no errors",
			wantError: false,
			specified: map[int64]bool{
				LvlInfo: true,
			},
			levelRatios: []decimal.Decimal{
				decimal.NewFromInt(0),   // Fatal
				decimal.NewFromInt(0),   // Error
				decimal.NewFromInt(0),   // Warn
				decimal.NewFromInt(100), // Info
				decimal.NewFromInt(0),   // Debug
				decimal.NewFromInt(0),   // Trace
			},
			expectedRanges:     []int64{0, 0, 0, 60000, 0, 0},
			expectedCountFatal: int64(0),
			expectedCountError: int64(0),
			expectedCountWarn:  int64(0),
			expectedCountInfo:  int64(2000),
			expectedCountDebug: int64(0),
			expectedCountTrace: int64(0),
		},
		{
			name:      "override, 10 pct errors, 22pct for others, no errors",
			wantError: false,
			specified: map[int64]bool{
				LvlError: true,
			},
			levelRatios: []decimal.Decimal{
				decimal.NewFromInt(0),  // Fatal
				decimal.NewFromInt(10), // Error
				decimal.NewFromInt(0),  // Warn
				decimal.NewFromInt(0),  // Info
				decimal.NewFromInt(0),  // Debug
				decimal.NewFromInt(0),  // Trace
			},
			expectedRanges:     []int64{0, 6000, 13500, 13500, 13500, 13500},
			expectedCountFatal: int64(0), // 2000-200 = 1800 / 4 = 450
			expectedCountError: int64(200),
			expectedCountWarn:  int64(450),
			expectedCountInfo:  int64(450),
			expectedCountDebug: int64(450),
			expectedCountTrace: int64(450),
		},
		{
			name:      "override, 13 pct errors, 17 warn, 5 info",
			wantError: false,
			specified: map[int64]bool{
				LvlError: true,
				LvlWarn:  true,
				LvlInfo:  true,
			},
			levelRatios: []decimal.Decimal{
				decimal.NewFromInt(0),  // Fatal
				decimal.NewFromInt(13), // Error
				decimal.NewFromInt(7),  // Warn
				decimal.NewFromInt(5),  // Info
				decimal.NewFromInt(0),  // Debug
				decimal.NewFromInt(0),  // Trace
			},
			expectedRanges:     []int64{0, 7800, 4200, 3000, 22500, 22500},
			expectedCountFatal: int64(0),
			expectedCountError: int64(260), // 2000-250=1740
			expectedCountWarn:  int64(140), // 1740-140=1500
			expectedCountInfo:  int64(100), // 1600-100=1500
			expectedCountDebug: int64(750), // 1500-750=750
			expectedCountTrace: int64(750),
		},
	}

	logLinesToGenerate := int(2000)

	delayBetweenLines := int64(0) // set delay to 0 for fastest execution

	// a rate 5% per every 1000 lines of output permits our tests to pass
	allowedDeviance := int64(50)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator, errs := New(tt.specified, tt.levelRatios)
			if tt.wantError {
				require.NotZero(t, len(errs), "missing expected validation errors")
				return
			} else {
				require.Len(t, errs, 0, "default validation failed")
			}

			// we don't actually need to see the lines, so ignore
			generator.Logger.Out = io.Discard

			ranges := generator.DeriveDistributionRanges()
			assert.Equal(t, tt.expectedRanges, ranges, "ranges are wrong")

			outputCounts := generator.Output(ranges, logLinesToGenerate, delayBetweenLines)
			// t.Logf("'%s' counts: %+v\n", tt.name, outputCounts)

			if tt.expectedCountFatal == 0 {
				_, ok := outputCounts[LvlFatal]
				assert.False(t, ok, "fatal messages found")
			} else {
				// fatal is a special case, we'll need to catch the exit for a proper test.
			}

			if tt.expectedCountError == 0 {
				_, ok := outputCounts[LvlError]
				assert.False(t, ok, "error messages found")
			} else {
				msgCountError := outputCounts[LvlError]
				assert.GreaterOrEqual(t, msgCountError, tt.expectedCountError-allowedDeviance,
					"error message count less than expected")
				assert.LessOrEqual(t, msgCountError, tt.expectedCountError+allowedDeviance,
					"error message count larger than expected")
			}

			if tt.expectedCountWarn == 0 {
				_, ok := outputCounts[LvlWarn]
				assert.False(t, ok, "warn messages found")
			} else {
				msgCountWarn := outputCounts[LvlWarn]
				assert.GreaterOrEqual(t, msgCountWarn, tt.expectedCountWarn-allowedDeviance,
					"warn message count less than expected")
				assert.LessOrEqual(t, msgCountWarn, tt.expectedCountWarn+allowedDeviance,
					"warn message count larger than expected")
			}

			if tt.expectedCountInfo == 0 {
				_, ok := outputCounts[LvlInfo]
				assert.False(t, ok, "info messages found")
			} else {
				msgCountInfo := outputCounts[LvlInfo]
				assert.GreaterOrEqual(t, msgCountInfo, tt.expectedCountInfo-allowedDeviance,
					"info message count less than expected")
				assert.LessOrEqual(t, msgCountInfo, tt.expectedCountInfo+allowedDeviance,
					"info message count larger than expected")
			}

			if tt.expectedCountDebug == 0 {
				_, ok := outputCounts[LvlDebug]
				assert.False(t, ok, "debug messages found")
			} else {
				msgCountDebug := outputCounts[LvlDebug]
				assert.GreaterOrEqual(t, msgCountDebug, tt.expectedCountDebug-allowedDeviance,
					"debug message count less than expected")
				assert.LessOrEqual(t, msgCountDebug, tt.expectedCountDebug+allowedDeviance,
					"debug message count larger than expected")
			}
		})
	}
}
