package main

import (
//	"bytes"
	"flag"
	"fmt"
	vegeta "github.com/senaduka/vegeta/lib"
	"log"
	"net/http"
	"strings"
	"time"
	"strconv"
)


type rateList []uint64


// String is the method to format the flag's value, part of the flag.Value interface.
// The String method's output will be used in diagnostics.
func (i *rateList) String() string {
	return fmt.Sprint(*i)
}

// Set is the method to set the flag value, part of the flag.Value interface.
// Set's argument is a string to be parsed to set the flag.
// It's a comma-separated list, so we split it.
func (i *rateList) Set(value string) error {
	
	for _, singleRate := range strings.Split(value, ",") {
		oneRate, err := strconv.ParseUint(singleRate, 10, 64)
		if err != nil {
			return err
		}
		*i = append(*i, oneRate)
	}
	return nil
}


// attack validates the attack arguments, sets up the
// required resources, launches the attack and writes the results
func rapidAttack(rate uint64, duration time.Duration, targets *vegeta.Targets , ordering, output string, header http.Header, previousResults vegeta.Results ) (vegeta.Results, error) {

    
	if rate == 0 {
		return nil, fmt.Errorf(errRatePrefix + "can't be zero")
	}

	if duration == 0 {
		return nil, fmt.Errorf(errDurationPrefix + "can't be zero")
	}


	targets.SetHeader(header)

	switch ordering {
	case "random":
		targets.Shuffle(time.Now().UnixNano())
	case "sequential":
		break
	default:
		return nil, fmt.Errorf(errOrderingPrefix+"`%s` is invalid", ordering)
	}

	

	log.Printf("Vegeta is attacking %d targets in %s order for %s with %d requests/sec...\n", len(*targets), ordering, duration, rate)
	results := vegeta.Attack(*targets, rate, duration)
	log.Println("Done!")

	return append(previousResults, results...), nil
}


func writeResults(results vegeta.Results , output string)  error {



	out, err := file(output, true)
	if err != nil {
		return fmt.Errorf(errOutputFilePrefix+"(%s): %s", output, err)
	}
	defer out.Close()

	log.Printf("Writing results to '%s'...", output)
	if err := results.Encode(out); err != nil {
		return err
	}
	return nil
}




func rapidCmd(args []string) command {
	fs := flag.NewFlagSet("rapid", flag.ExitOnError)
	targetsf := fs.String("targets", "stdin", "Targets file")
	ordering := fs.String("ordering", "random", "Attack ordering [sequential, random]")
	duration := fs.Duration("duration", 10*time.Second, "Duration of the attack on every rate")
	output := fs.String("output", "stdout", "Output file")
	hdrs := headers{Header: make(http.Header)}
	fs.Var(hdrs, "header", "Targets request header")
	var rateFlag rateList = []uint64{}
	fs.Var(&rateFlag, "rates", "One or more rates, comma separated in requests per second")
	fs.Parse(args)

	in, err := file(*targetsf, false)
	if err != nil {
		return nil
	}
	defer in.Close()
	targets, err := vegeta.NewTargetsFrom(in)
	if err != nil {
		return nil
	}



	return func() error {

		results := make(vegeta.Results,0)
		var err error = nil
		
		for _, rate := range rateFlag {

			
				if results, err = rapidAttack(rate, *duration, &targets, *ordering, *output, hdrs.Header, results); err != nil {
					return err 
				}
		}

		if err = writeResults(results, *output); err != nil {
			return err
		}

		return nil
	}
}