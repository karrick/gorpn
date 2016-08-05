package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/karrick/godag"
	"github.com/karrick/gorpn"
)

type Evaluator interface {
	Evaluate(map[string]interface{}) (float64, error)
}

type Datum struct {
	When time.Time
	What float64
}

type Series []Datum

func (s *Series) Evaluate(when time.Time) float64 {
	list := *s
	lo := 0
	hi := len(list)
	for lo <= hi {
		index := (lo + hi) / 2
		value := list[index]
		if when.Before(value.When) {
			hi = index - 1
		} else if when.After(value.When) {
			lo = index + 1
		} else {
			return value.What
		}
	}
	return math.NaN()
}

// func (d *Datum) Evaluate(_ map[string]interface{}) (float64, error) {
// }

func main() {

	// make a few defs

	data := map[string]interface{}{
		"age": 5,
		"bar": 5,
	}

	// make a few cdefs

	cdefs := make(map[string]*gorpn.Expression)

	exp, err := gorpn.New("age,12,*")
	if err != nil {
		panic(err)
	}
	cdefs["month"] = exp

	exp, err = gorpn.New("bar,42,/")
	if err != nil {
		panic(err)
	}
	cdefs["whole"] = exp

	//

	dag := godag.New()
	bindings := make(map[string]interface{})

	// need to load defs because cdefs refer to them
	for label := range data {
		bindings[label] = math.NaN()
		dag.Insert(label, nil)
	}

	for label, exp := range cdefs {
		var deps []string
		_, err := exp.Evaluate(bindings)
		if err != nil {
			// get the open bindings
			if d, ok := err.(gorpn.ErrOpenBindings); ok {
				deps = make([]string, len(d))
				copy(deps, []string(d))
			}
		}
		dag.Insert(label, deps)
	}

	ordered, err := dag.Order()
	if err != nil {
		// could be missing series

		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	fmt.Printf("ordered: %+v\n", ordered)
}
