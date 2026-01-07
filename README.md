# Overview

**LogGenerator** is a small Golang module which can generate a stream of log messages according to a specified set of desired ratios.

The motivation stems from wanting to be able to work with dynamic log data, but not always having access or wanting to impact actual deployed services.

The logic accepts a list of default values to work with and allows for overwriting these from user input, but needs to be told which values should be left alone so that the others can have their percentages increased in the case where the caller might only specify a few levels and allow the logic to figure out the rest.

### Usage

You can provide the desired ratios of level that should add up to 100 percent like so:

```go
    import "github.com/robojandro/loggenerator"

    ...

    // set up some sensible defaults
	levelRatios := []decimal.Decimal{
	▏   decimal.NewFromInt(0),  // Fatal
	▏   decimal.NewFromInt(10), // Error
	▏   decimal.NewFromInt(20), // Warn
	▏   decimal.NewFromInt(50), // Info
	▏   decimal.NewFromInt(20), // Debug
	▏   decimal.NewFromInt(0),  // Trace
	}

	// specified is required to flag if any of the above defaults were overriden
	// required the module to re-calculate how the ratio percentages
	// if there were no overrides it can be empty
	specified := make(map[int64]bool, 6)

	// if there were overrides, such if one wanted all Info level messages it
	//  would need to state which of the values were stated as overrides
	//		specified[loggenerator.LvlInfo] = true
	
	generator, errs := loggenerator.New(specified, levelRatios)
	if len(errs) != 0 {
		fmt.Printf("Invalid ratios set: %+v\nErrors: %+v\n", levelRatios, errs)
		os.Exit(1)
	}

	// derive the linear range of values per the level percentages
	// needed for the randomized output generation
	ranges := generator.DeriveDistributionRanges()

	// how many lines to generate
	outputLimit := int(100)

	// 10 milliseconds is very fast, use values for 100 or higher to be able to distinguish
	// the output as it happens
	delay := int64(10) 

	// actually generate the output using the limit and delay parameters
	outputCounts := generator.Output(ranges, outputLimit, delay)

	// print the actual total of log lines that were output
	fmt.Printf("output counts: %+v\n", outputCounts)
```

Setting Fatal to anything but zero will result in the eventual early termination of the output.

### Direct Dependencies

Shopify's Decimal package to handle the conversion from between numerical types more elegantly 
- github.com/shopspring/decimal

Sirupsen's structured logger which supports lots of extendibility
- github.com/sirupsen/logrus

Stretchr's test assertion and require packages which I much prefer over the vanilla `testing` package
- github.com/stretchr/testify/assert

### See Also

Service that uses this module to simulate a deployed Kubernetes service
- github.com/robojandro/loggingmockservice
