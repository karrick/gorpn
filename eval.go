package gorpn

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const DefaultDelimeter = ","

// type arityTuple [3]int
type arityTuple struct {
	popCount, floatOffset, floatCount, nonOperatorOffset, nonOperatorCount int
}

// arity resolves to the number of items an operation must pop, and
// how many of those must be floats
var arity = map[string]arityTuple{
	"%":       {2, 2, 2, 0, 0},
	"*":       {2, 2, 2, 0, 0},
	"+":       {2, 2, 2, 0, 0},
	"-":       {2, 2, 2, 0, 0},
	"/":       {2, 2, 2, 0, 0},
	"ABS":     {1, 1, 1, 0, 0},
	"ADDNAN":  {2, 2, 2, 0, 0},
	"ATAN":    {1, 1, 1, 0, 0},
	"ATAN2":   {2, 2, 2, 0, 0},
	"AVG":     {1, 1, 1, 0, 0}, // other operands must be floats
	"CEIL":    {1, 1, 1, 0, 0},
	"COPY":    {1, 1, 1, 0, 0}, // other operands cannot be operators
	"COS":     {1, 1, 1, 0, 0},
	"DEG2RAD": {1, 1, 1, 0, 0},
	"DEPTH":   {0, 0, 0, 0, 0},
	"DUP":     {1, 0, 0, 1, 1}, // equivalent to: 1,COPY
	"EQ":      {2, 0, 0, 2, 2},
	"EXC":     {2, 0, 0, 2, 2}, // equivalent to: 2,REV
	"EXP":     {1, 1, 1, 0, 0},
	"FLOOR":   {1, 1, 1, 0, 0},
	"GE":      {2, 0, 0, 2, 2},
	"GT":      {2, 0, 0, 2, 2},
	"IF":      {3, 3, 1, 2, 2}, // a,b,c,IF
	"INDEX":   {1, 1, 1, 0, 0}, // other operands cannot be operators
	"ISINF":   {1, 1, 1, 0, 0},
	"LE":      {2, 0, 0, 2, 2},
	"LIMIT":   {3, 3, 3, 0, 0},
	"LOG":     {1, 1, 1, 0, 0},
	"LT":      {2, 0, 0, 2, 2},
	"MAX":     {2, 0, 0, 2, 2},
	"MAXNAN":  {2, 0, 0, 2, 2},
	"MIN":     {2, 0, 0, 2, 2},
	"MINNAN":  {2, 0, 0, 2, 2},
	"NE":      {2, 0, 0, 2, 2},
	"POP":     {1, 0, 0, 0, 0},
	"RAD2DEG": {1, 1, 1, 0, 0},
	"REV":     {1, 1, 1, 0, 0}, // other operands cannot be operators
	"ROLL":    {2, 2, 2, 0, 0}, // n,m,ROLL (rotate the top n elements of the stack by m)
	"SIN":     {1, 1, 1, 0, 0},
	"SORT":    {1, 1, 1, 0, 0}, // other operands must be floats
	"SQRT":    {1, 1, 1, 0, 0},
	"UN":      {1, 1, 1, 0, 0},
}

// ExpectedFloat error is returned if a different data type is
// discovered where a float64 value is required.
type ExpectedFloat struct {
	v interface{}
}

// Error returns the error string representation for ExpectedFloat errors.
func (e ExpectedFloat) Error() string {
	return fmt.Sprintf("expected float: %T", e.v)
}

// ErrOpenVariables error is returned when one or more open variables
// remain when evaluating a RPN Expression.
type ErrOpenVariables []string

// Error returns the error string representation for ErrOpenVariables
// errors.
func (e ErrOpenVariables) Error() string {
	return "open variables: " + strings.Join(e, ",")
}

// ErrSyntax error is returned if the specified RPN expression
// does not evaluate because of a syntax error.
type ErrSyntax struct {
	Message string
	Err     error
}

// Error returns the error string representation for ErrSyntax errors.
func (e ErrSyntax) Error() string {
	if e.Err == nil {
		return "syntax error " + e.Message
	}
	return "syntax error " + e.Message + ": " + e.Err.Error()
}

func newErrSyntax(a ...interface{}) ErrSyntax {
	var err error
	var format, message string
	var ok bool
	if len(a) == 0 {
		return ErrSyntax{"no reason given", nil}
	}
	// if last item is error: save it
	if err, ok = a[len(a)-1].(error); ok {
		a = a[:len(a)-1] // pop it
	}
	// if items left, first ought to be format string
	if len(a) > 0 {
		if format, ok = a[0].(string); ok {
			a = a[1:] // unshift
			message = fmt.Sprintf(format, a...)
		}
	}
	if message != "" {
		message = ": " + message
	}
	return ErrSyntax{message, err}
}

// Expression represents a RPN expression.
type Expression struct {
	delimeter     string
	usesTime      bool
	tokens        []interface{} // components of the expression
	openVariables []string      // duplicates may occur
	// work area
	scratchSize int           // how much work area this needs
	scratchHead int           // index of top of scratch and isFloat slices
	scratch     []interface{} // work area where calculations are done
	isFloat     []bool        // true iff corresponding scratch item is a float64
}

// New returns a new RPN Expression based on some expression.
func New(someExpression string, setters ...ExpressionSetter) (*Expression, error) {
	if someExpression == "" {
		return nil, ErrSyntax{"empty expression", nil}
	}
	e := &Expression{delimeter: DefaultDelimeter}
	for _, setter := range setters {
		if err := setter(e); err != nil {
			return nil, err
		}
	}
	tokens := strings.Split(someExpression, e.delimeter)
	e.scratchSize = len(tokens)

	e.tokens = make([]interface{}, e.scratchSize)
	for idx, token := range tokens {
		switch token {
		case "NOW":
			e.usesTime = true
		case "DUP":
			e.scratchSize++
		}
		e.tokens[idx] = token
	}
	// scratchSize may be larger than it was before above loop
	e.scratch = make([]interface{}, e.scratchSize)
	e.isFloat = make([]bool, e.scratchSize)
	return e.Partial(make(map[string]float64))
}

// ExpressionSetter represents a function that modifies an RPN
// Expression.
type ExpressionSetter func(*Expression) error

// Delimeter allows changing the expected delimeter for an RPN
// Expression from the default delimeter, the comma.
func Delimeter(someByte string) ExpressionSetter {
	return func(e *Expression) error {
		e.delimeter = someByte
		return nil
	}
}

// String returns the string representation of an Expression.
func (e Expression) String() string {
	strs := make([]string, len(e.tokens))
	for idx, v := range e.tokens {
		switch v.(type) {
		case float64:
			switch {
			case math.IsNaN(v.(float64)):
				// strs[idx] = "NaN" // would prefer this
				strs[idx] = "UNKN" // don't like this
			case math.IsInf(v.(float64), 1):
				strs[idx] = "INF"
			case math.IsInf(v.(float64), -1):
				strs[idx] = "NEGINF"
			default:
				strs[idx] = fmt.Sprint(v)
			}
		case string:
			strs[idx] = v.(string)
		default:
			strs[idx] = fmt.Sprint(v)
		}
	}
	return strings.Join(strs, e.delimeter)
}

// Partial creates a new Expression by partial application of the
// parameter bindings. With the additional bindings, it attempts to
// further simplify the expression.
func (e *Expression) Partial(bindings map[string]float64) (*Expression, error) {
	exp := &Expression{
		delimeter:   e.delimeter,
		tokens:      make([]interface{}, len(e.tokens)),
		scratchSize: e.scratchSize,
		scratch:     make([]interface{}, e.scratchSize),
		isFloat:     make([]bool, e.scratchSize),
	}
	copy(exp.tokens, e.tokens)

	err := exp.simplify(bindings)
	switch err.(type) {
	case nil:
		// promote our work area to our new stored program
		exp.tokens = exp.tokens[:exp.scratchHead] // shrink tokens first
		copy(exp.tokens, exp.scratch)

		exp.usesTime = e.usesTime // set after simplify() to prevent calculating NOW
		return exp, nil
	}
	return nil, err
}

// Evaluate evaluates the Expression after applying the parameter
// bindings.
func (e *Expression) Evaluate(bindings map[string]float64) (float64, error) {
	err := e.simplify(bindings)
	if err != nil {
		return 0, err
	}
	if e.openVariables != nil {
		return 0, ErrOpenVariables(e.openVariables)
	}
	if e.scratchHead != 1 {
		return 0, newErrSyntax("extra parameters: %v", e.scratch)
	}
	result, ok := e.scratch[0].(float64)
	if !ok {
		return 0, ExpectedFloat{e.scratch[0]}
	}
	return result, nil
}

// Valid returns true iff Expression is valid RPN.
func (e Expression) Valid() bool {
	return e.valid(make(map[string]float64))
}

func (e Expression) valid(bindings map[string]float64) bool {
	err := e.simplify(bindings)
	if err != nil {
		return false
	}
	if e.openVariables != nil {
		for _, item := range e.openVariables {
			bindings[item] = 0
		}
		return e.valid(bindings)
	}
	if e.scratchHead != 1 {
		return false
	}
	return e.isFloat[0]
}

func (e *Expression) simplify(bindings map[string]float64) error {
	// with a fresh start comes fresh workspace
	e.scratchHead = 0
	e.openVariables = nil

	// heisenberg principle, realized: it takes take to observe the time, so do it only once
	var now interface{}
	var nowFloat bool
	if e.usesTime {
		now = float64(time.Now().Unix())
		nowFloat = true
	} else {
		now = "NOW"
	}

	// variables outside of loop to reduce allocations
	var err error
	var cannotSimplify, isFloat, ok, stackUpdated, firstNaN, secondNaN bool
	var total, value float64
	var argIdx, count, indexOfFirstArg, itemIdx, tokIdx, used int
	var opArity arityTuple
	var result, tok interface{}

	// tokens is our stored program, and scratch is our work area
	for tokIdx, tok = range e.tokens {
		switch tok.(type) {
		case float64:
			e.scratch[e.scratchHead] = tok
			e.isFloat[e.scratchHead] = true
			e.scratchHead++
		case string:
			switch token := tok.(string); token {
			case "MINUTE":
				e.scratch[e.scratchHead] = float64(60)
				e.isFloat[e.scratchHead] = true
				e.scratchHead++
			case "HOUR":
				e.scratch[e.scratchHead] = float64(3600)
				e.isFloat[e.scratchHead] = true
				e.scratchHead++
			case "DAY":
				e.scratch[e.scratchHead] = float64(86400)
				e.isFloat[e.scratchHead] = true
				e.scratchHead++
			case "WEEK":
				e.scratch[e.scratchHead] = float64(604800)
				e.isFloat[e.scratchHead] = true
				e.scratchHead++
			case "UNKN":
				e.scratch[e.scratchHead] = math.NaN()
				e.isFloat[e.scratchHead] = true
				e.scratchHead++
			case "INF":
				e.scratch[e.scratchHead] = math.Inf(1)
				e.isFloat[e.scratchHead] = true
				e.scratchHead++
			case "NEGINF":
				e.scratch[e.scratchHead] = math.Inf(-1)
				e.isFloat[e.scratchHead] = true
				e.scratchHead++
			case "NOW":
				e.scratch[e.scratchHead] = now
				e.isFloat[e.scratchHead] = nowFloat
				e.scratchHead++
			case "":
				return newErrSyntax("empty token")
			default:
				if opArity, ok = arity[token]; ok {
					stackUpdated = false
					cannotSimplify = false

					// ??? popCount = floatCount + nonOperatorCount

					if e.scratchHead < opArity.popCount {
						return newErrSyntax("not enough parameters: operator %s requires %d operands", token, opArity.popCount)
					}
					indexOfFirstArg = e.scratchHead - opArity.popCount

					// fmt.Println("FLOAT CHECK: e.tokens:", e.tokens, "e.scratch:", e.scratch[:e.head], "floatOffset:", opArity.floatOffset, "floatCount:", opArity.floatCount)
					for argIdx = e.scratchHead - opArity.floatOffset; argIdx < e.scratchHead-opArity.floatOffset+opArity.floatCount; argIdx++ {
						if _, isFloat = e.scratch[argIdx].(float64); !isFloat {
							// fmt.Println("found non float:", e.scratch[argIdx])
							cannotSimplify = true
							break
						}
					}

					// fmt.Println("NOT OPERATOR CHECK: e.tokens:", e.tokens, "e.scratch:", e.scratch[:e.head], "opArity.nonOperatorOffset:", opArity.nonOperatorOffset, "opArity.nonOperatorCount:", opArity.nonOperatorCount)
					for argIdx = e.scratchHead - opArity.nonOperatorOffset; argIdx < e.scratchHead-opArity.nonOperatorOffset+opArity.nonOperatorCount; argIdx++ {
						// fmt.Println("e.tokens:", e.tokens, "argIdx:", argIdx)
						if !e.isFloat[argIdx] {
							result = e.scratch[argIdx]
							if _, ok = arity[result.(string)]; ok {
								// fmt.Println("found operator:", e.scratch[argIdx])
								cannotSimplify = true
								break
							}
						}
					}
					if !cannotSimplify {
						switch token {
						case "+":
							result = e.scratch[indexOfFirstArg].(float64) + e.scratch[indexOfFirstArg+1].(float64)
						case "-":
							result = e.scratch[indexOfFirstArg].(float64) - e.scratch[indexOfFirstArg+1].(float64)
						case "*":
							result = e.scratch[indexOfFirstArg].(float64) * e.scratch[indexOfFirstArg+1].(float64)
						case "/":
							result = e.scratch[indexOfFirstArg].(float64) / e.scratch[indexOfFirstArg+1].(float64)
						case "%":
							result = math.Mod(e.scratch[indexOfFirstArg].(float64), e.scratch[indexOfFirstArg+1].(float64))
						case "ADDNAN":
							firstNaN = math.IsNaN(e.scratch[indexOfFirstArg].(float64))
							secondNaN = math.IsNaN(e.scratch[indexOfFirstArg+1].(float64))
							if !firstNaN && !secondNaN {
								result = e.scratch[indexOfFirstArg].(float64) + e.scratch[indexOfFirstArg+1].(float64)
							} else if !firstNaN {
								result = e.scratch[indexOfFirstArg]
							} else {
								result = e.scratch[indexOfFirstArg+1]
							}
						case "MAX":
							if e.isFloat[indexOfFirstArg] && e.isFloat[indexOfFirstArg+1] {
								if math.IsNaN(e.scratch[indexOfFirstArg].(float64)) {
									result = e.scratch[indexOfFirstArg]
								} else if math.IsNaN(e.scratch[indexOfFirstArg+1].(float64)) {
									result = e.scratch[indexOfFirstArg+1]
								} else {
									result = math.Max(e.scratch[indexOfFirstArg+1].(float64), e.scratch[indexOfFirstArg].(float64))
								}
							} else if !e.isFloat[indexOfFirstArg] && !e.isFloat[indexOfFirstArg+1] {
								if e.scratch[indexOfFirstArg].(string) == e.scratch[indexOfFirstArg+1].(string) {
									result = e.scratch[indexOfFirstArg]
								} else {
									cannotSimplify = true
								}
							} else if e.isFloat[indexOfFirstArg] && math.IsNaN(e.scratch[indexOfFirstArg].(float64)) {
								result = e.scratch[indexOfFirstArg]
							} else if e.isFloat[indexOfFirstArg+1] && math.IsNaN(e.scratch[indexOfFirstArg+1].(float64)) {
								result = e.scratch[indexOfFirstArg+1]
							} else {
								cannotSimplify = true
							}
						case "MAXNAN":
							if e.isFloat[indexOfFirstArg] && e.isFloat[indexOfFirstArg+1] {
								if math.IsNaN(e.scratch[indexOfFirstArg].(float64)) {
									result = e.scratch[indexOfFirstArg+1]
								} else if math.IsNaN(e.scratch[indexOfFirstArg+1].(float64)) {
									result = e.scratch[indexOfFirstArg]
								} else {
									result = math.Max(e.scratch[indexOfFirstArg+1].(float64), e.scratch[indexOfFirstArg].(float64))
								}
							} else if !e.isFloat[indexOfFirstArg] && !e.isFloat[indexOfFirstArg+1] {
								if e.scratch[indexOfFirstArg].(string) == e.scratch[indexOfFirstArg+1].(string) {
									result = e.scratch[indexOfFirstArg]
								} else {
									cannotSimplify = true
								}
							} else if e.isFloat[indexOfFirstArg] && math.IsNaN(e.scratch[indexOfFirstArg].(float64)) {
								result = e.scratch[indexOfFirstArg+1]
							} else if e.isFloat[indexOfFirstArg+1] && math.IsNaN(e.scratch[indexOfFirstArg+1].(float64)) {
								result = e.scratch[indexOfFirstArg]
							} else {
								cannotSimplify = true
							}
						case "MIN":
							if e.isFloat[indexOfFirstArg] && e.isFloat[indexOfFirstArg+1] {
								if math.IsNaN(e.scratch[indexOfFirstArg].(float64)) {
									result = e.scratch[indexOfFirstArg]
								} else if math.IsNaN(e.scratch[indexOfFirstArg+1].(float64)) {
									result = e.scratch[indexOfFirstArg+1]
								} else {
									result = math.Min(e.scratch[indexOfFirstArg+1].(float64), e.scratch[indexOfFirstArg].(float64))
								}
							} else if !e.isFloat[indexOfFirstArg] && !e.isFloat[indexOfFirstArg+1] {
								if e.scratch[indexOfFirstArg].(string) == e.scratch[indexOfFirstArg+1].(string) {
									result = e.scratch[indexOfFirstArg]
								} else {
									cannotSimplify = true
								}
							} else if e.isFloat[indexOfFirstArg] && math.IsNaN(e.scratch[indexOfFirstArg].(float64)) {
								result = e.scratch[indexOfFirstArg]
							} else if e.isFloat[indexOfFirstArg+1] && math.IsNaN(e.scratch[indexOfFirstArg+1].(float64)) {
								result = e.scratch[indexOfFirstArg+1]
							} else {
								cannotSimplify = true
							}
						case "MINNAN":
							if e.isFloat[indexOfFirstArg] && e.isFloat[indexOfFirstArg+1] {
								if math.IsNaN(e.scratch[indexOfFirstArg].(float64)) {
									result = e.scratch[indexOfFirstArg+1]
								} else if math.IsNaN(e.scratch[indexOfFirstArg+1].(float64)) {
									result = e.scratch[indexOfFirstArg]
								} else {
									result = math.Min(e.scratch[indexOfFirstArg+1].(float64), e.scratch[indexOfFirstArg].(float64))
								}
							} else if !e.isFloat[indexOfFirstArg] && !e.isFloat[indexOfFirstArg+1] {
								if e.scratch[indexOfFirstArg].(string) == e.scratch[indexOfFirstArg+1].(string) {
									result = e.scratch[indexOfFirstArg]
								} else {
									cannotSimplify = true
								}
							} else if e.isFloat[indexOfFirstArg] && math.IsNaN(e.scratch[indexOfFirstArg].(float64)) {
								result = e.scratch[indexOfFirstArg+1]
							} else if e.isFloat[indexOfFirstArg+1] && math.IsNaN(e.scratch[indexOfFirstArg+1].(float64)) {
								result = e.scratch[indexOfFirstArg]
							} else {
								cannotSimplify = true
							}
						case "IF":
							// A,B,C,IF ==> A ? B : C
							if e.isFloat[indexOfFirstArg] {
								if e.scratch[indexOfFirstArg].(float64) < 0 || e.scratch[indexOfFirstArg].(float64) > 0 {
									result = e.scratch[indexOfFirstArg+1]
								} else {
									result = e.scratch[indexOfFirstArg+2]
								}
							} else {
								cannotSimplify = true
							}
						case "LIMIT":
							if math.IsNaN(e.scratch[indexOfFirstArg].(float64)) || math.IsNaN(e.scratch[indexOfFirstArg+1].(float64)) || math.IsNaN(e.scratch[indexOfFirstArg+2].(float64)) {
								result = math.NaN()
							} else if math.IsInf(e.scratch[indexOfFirstArg].(float64), -1) || math.IsInf(e.scratch[indexOfFirstArg+1].(float64), -1) || math.IsInf(e.scratch[indexOfFirstArg+2].(float64), -1) {
								result = math.NaN()
							} else if !(e.scratch[indexOfFirstArg].(float64) < e.scratch[indexOfFirstArg+1].(float64) || e.scratch[indexOfFirstArg].(float64) > e.scratch[indexOfFirstArg+2].(float64)) {
								result = e.scratch[indexOfFirstArg]
							} else {
								result = math.NaN()
							}
						case "EQ":
							if e.isFloat[indexOfFirstArg] && e.isFloat[indexOfFirstArg+1] {
								if e.scratch[indexOfFirstArg].(float64) == e.scratch[indexOfFirstArg+1].(float64) {
									result = float64(1)
								} else {
									result = float64(0)
								}
							} else if !e.isFloat[indexOfFirstArg] && !e.isFloat[indexOfFirstArg+1] {
								if e.scratch[indexOfFirstArg].(string) == e.scratch[indexOfFirstArg+1].(string) {
									result = float64(1)
								} else {
									cannotSimplify = true
								}
							} else {
								cannotSimplify = true
							}
						case "NE":
							if e.isFloat[indexOfFirstArg] && e.isFloat[indexOfFirstArg+1] {
								if e.scratch[indexOfFirstArg].(float64) != e.scratch[indexOfFirstArg+1].(float64) {
									result = float64(1)
								} else {
									result = float64(0)
								}
							} else if !e.isFloat[indexOfFirstArg] && !e.isFloat[indexOfFirstArg+1] {
								if e.scratch[indexOfFirstArg].(string) == e.scratch[indexOfFirstArg+1].(string) {
									result = float64(0)
								} else {
									cannotSimplify = true
								}
							} else {
								cannotSimplify = true
							}
						case "GE":
							if e.isFloat[indexOfFirstArg] && e.isFloat[indexOfFirstArg+1] {
								if e.scratch[indexOfFirstArg].(float64) >= e.scratch[indexOfFirstArg+1].(float64) {
									result = float64(1)
								} else {
									result = float64(0)
								}
							} else if !e.isFloat[indexOfFirstArg] && !e.isFloat[indexOfFirstArg+1] {
								if e.scratch[indexOfFirstArg].(string) == e.scratch[indexOfFirstArg+1].(string) {
									result = float64(1)
								} else {
									cannotSimplify = true
								}
							} else {
								cannotSimplify = true
							}
						case "LE":
							if e.isFloat[indexOfFirstArg] && e.isFloat[indexOfFirstArg+1] {
								if e.scratch[indexOfFirstArg].(float64) <= e.scratch[indexOfFirstArg+1].(float64) {
									result = float64(1)
								} else {
									result = float64(0)
								}
							} else if !e.isFloat[indexOfFirstArg] && !e.isFloat[indexOfFirstArg+1] {
								if e.scratch[indexOfFirstArg].(string) == e.scratch[indexOfFirstArg+1].(string) {
									result = float64(1)
								} else {
									cannotSimplify = true
								}
							} else {
								cannotSimplify = true
							}
						case "GT":
							if e.isFloat[indexOfFirstArg] && e.isFloat[indexOfFirstArg+1] {
								if e.scratch[indexOfFirstArg].(float64) > e.scratch[indexOfFirstArg+1].(float64) {
									result = float64(1)
								} else {
									result = float64(0)
								}
							} else if !e.isFloat[indexOfFirstArg] && !e.isFloat[indexOfFirstArg+1] {
								if e.scratch[indexOfFirstArg].(string) == e.scratch[indexOfFirstArg+1].(string) {
									result = float64(0)
								} else {
									cannotSimplify = true
								}
							} else {
								cannotSimplify = true
							}
						case "LT":
							if e.isFloat[indexOfFirstArg] && e.isFloat[indexOfFirstArg+1] {
								if e.scratch[indexOfFirstArg].(float64) < e.scratch[indexOfFirstArg+1].(float64) {
									result = float64(1)
								} else {
									result = float64(0)
								}
							} else if !e.isFloat[indexOfFirstArg] && !e.isFloat[indexOfFirstArg+1] {
								if e.scratch[indexOfFirstArg].(string) == e.scratch[indexOfFirstArg+1].(string) {
									result = float64(0)
								} else {
									cannotSimplify = true
								}
							} else {
								cannotSimplify = true
							}
						case "EXC":
							e.scratch[indexOfFirstArg], e.scratch[indexOfFirstArg+1] = e.scratch[indexOfFirstArg+1], e.scratch[indexOfFirstArg]
							e.isFloat[indexOfFirstArg], e.isFloat[indexOfFirstArg+1] = e.isFloat[indexOfFirstArg+1], e.isFloat[indexOfFirstArg]
							stackUpdated = true
						case "DEPTH":
							e.scratch[e.scratchHead] = e.scratchHead
							e.isFloat[e.scratchHead] = true
							e.scratchHead++
							stackUpdated = true
						case "COPY":
							if math.IsNaN(e.scratch[indexOfFirstArg].(float64)) || math.IsInf(e.scratch[indexOfFirstArg].(float64), 1) || math.IsInf(e.scratch[indexOfFirstArg].(float64), -1) || e.scratch[indexOfFirstArg].(float64) <= 0 {
								return newErrSyntax("%s operator requires positive finite integer: %v", token, e.scratch[indexOfFirstArg])
							}
							count = int(e.scratch[indexOfFirstArg].(float64))
							if count > e.scratchHead-1 {
								return newErrSyntax("%s %d items, but only %d on stack", token, count, e.scratchHead-1)
							}
							for argIdx = indexOfFirstArg - count; argIdx < indexOfFirstArg; argIdx++ {
								if !e.isFloat[argIdx] {
									if _, ok = arity[e.scratch[argIdx].(string)]; ok {
										cannotSimplify = true
										break
									}
								}
							}
							if !cannotSimplify {
								e.scratchHead--
								if e.scratchHead-1+count > cap(e.scratch) {
									// COPY requires larger scratch and isFloat slices
									scratch := make([]interface{}, e.scratchHead+count)
									copy(scratch, e.scratch)
									e.scratch = scratch
									isFloat := make([]bool, e.scratchHead+count)
									copy(isFloat, e.isFloat)
									e.isFloat = isFloat
								}
								for argIdx = indexOfFirstArg - count; argIdx < indexOfFirstArg; argIdx++ {
									e.scratch[e.scratchHead] = e.scratch[argIdx]
									e.isFloat[e.scratchHead] = e.isFloat[argIdx]
									e.scratchHead++
								}
								stackUpdated = true
							}
						case "INDEX":
							if math.IsNaN(e.scratch[indexOfFirstArg].(float64)) || math.IsInf(e.scratch[indexOfFirstArg].(float64), 1) || math.IsInf(e.scratch[indexOfFirstArg].(float64), -1) || e.scratch[indexOfFirstArg].(float64) <= 0 {
								return newErrSyntax("%s operator requires positive finite integer: %v", token, e.scratch[indexOfFirstArg])
							}
							count = int(e.scratch[indexOfFirstArg].(float64))
							if count > e.scratchHead-1 {
								return newErrSyntax("%s %d items, but only %d on stack", token, count, e.scratchHead-1)
							}
							for argIdx = indexOfFirstArg - count; argIdx < indexOfFirstArg; argIdx++ {
								if !e.isFloat[argIdx] {
									if _, ok = arity[e.scratch[argIdx].(string)]; ok {
										cannotSimplify = true
										break
									}
								}
							}
							if !cannotSimplify {
								e.scratch[e.scratchHead-1] = e.scratch[e.scratchHead-count-1]
								e.isFloat[e.scratchHead-1] = e.isFloat[e.scratchHead-count-1]
								stackUpdated = true
							}
						case "DUP":
							e.scratch[e.scratchHead] = e.scratch[e.scratchHead-1]
							e.isFloat[e.scratchHead] = e.isFloat[e.scratchHead-1]
							e.scratchHead++
							stackUpdated = true
						case "POP":
							e.scratchHead--
							stackUpdated = true
						case "ABS":
							result = math.Abs(e.scratch[indexOfFirstArg].(float64))
						case "CEIL":
							result = math.Ceil(e.scratch[indexOfFirstArg].(float64))
						case "FLOOR":
							result = math.Floor(e.scratch[indexOfFirstArg].(float64))
						case "ISINF":
							if math.IsInf(e.scratch[indexOfFirstArg].(float64), 1) || math.IsInf(e.scratch[indexOfFirstArg].(float64), -1) {
								result = float64(1)
							} else {
								result = float64(0)
							}
						case "UN":
							if math.IsNaN(e.scratch[indexOfFirstArg].(float64)) {
								result = float64(1)
							} else {
								result = float64(0)
							}
						case "DEG2RAD":
							result = e.scratch[indexOfFirstArg].(float64) * math.Pi / 180
						case "RAD2DEG":
							result = e.scratch[indexOfFirstArg].(float64) * 180 / math.Pi
						case "ATAN":
							result = math.Atan(e.scratch[indexOfFirstArg].(float64))
						case "ATAN2":
							result = math.Atan2(e.scratch[indexOfFirstArg+1].(float64), e.scratch[indexOfFirstArg].(float64))
						case "COS":
							result = math.Cos(e.scratch[indexOfFirstArg].(float64))
						case "SIN":
							result = math.Sin(e.scratch[indexOfFirstArg].(float64))
						case "LOG":
							result = math.Log(e.scratch[indexOfFirstArg].(float64))
						case "EXP":
							result = math.Exp(e.scratch[indexOfFirstArg].(float64))
						case "SQRT":
							result = math.Sqrt(e.scratch[indexOfFirstArg].(float64))
						case "AVG":
							if math.IsNaN(e.scratch[indexOfFirstArg].(float64)) || math.IsInf(e.scratch[indexOfFirstArg].(float64), 1) || math.IsInf(e.scratch[indexOfFirstArg].(float64), -1) || e.scratch[indexOfFirstArg].(float64) <= 0 {
								return newErrSyntax("%s operator requires positive finite integer: %v", token, e.scratch[indexOfFirstArg])
							}
							count = int(e.scratch[indexOfFirstArg].(float64))
							if count > e.scratchHead-1 {
								return newErrSyntax("%s %d items, but only %d on stack", token, count, e.scratchHead-1)
							}
							total = 0
							used = 0
							for argIdx = indexOfFirstArg - count; argIdx < indexOfFirstArg; argIdx++ {
								if !e.isFloat[argIdx] {
									cannotSimplify = true
									break
								}
								if !math.IsNaN(e.scratch[argIdx].(float64)) {
									total += e.scratch[argIdx].(float64)
									used++
								}
							}
							if !cannotSimplify {
								e.scratchHead -= (count + opArity.popCount)
								e.scratch[e.scratchHead] = total / float64(used)
								e.isFloat[e.scratchHead] = true
								e.scratchHead++
								stackUpdated = true
							}
						case "REV":
							if math.IsNaN(e.scratch[indexOfFirstArg].(float64)) || math.IsInf(e.scratch[indexOfFirstArg].(float64), 1) || math.IsInf(e.scratch[indexOfFirstArg].(float64), -1) || e.scratch[indexOfFirstArg].(float64) <= 0 {
								return newErrSyntax("%s operator requires positive finite integer: %v", token, e.scratch[indexOfFirstArg])
							}
							count = int(e.scratch[indexOfFirstArg].(float64))
							if count > e.scratchHead-1 {
								return newErrSyntax("%s %d items, but only %d on stack", token, count, e.scratchHead-1)
							}
							// cannot rev if any are operators
							for argIdx = indexOfFirstArg - count; argIdx < indexOfFirstArg; argIdx++ {
								if !e.isFloat[argIdx] {
									if _, ok = arity[e.scratch[argIdx].(string)]; ok {
										cannotSimplify = true
										break
									}
								}
							}
							if !cannotSimplify {
								items := make([]interface{}, count)
								e.scratchHead-- // drop the count
								copy(items, e.scratch[e.scratchHead-count:])
								itemIdx = count - 1
								for argIdx = indexOfFirstArg - count; argIdx < indexOfFirstArg; argIdx++ {
									// overwrite other elements
									_, isFloat = items[itemIdx].(float64)
									e.scratch[argIdx] = items[itemIdx]
									e.isFloat[argIdx] = isFloat
									itemIdx--
								}
								stackUpdated = true
							}
						case "ROLL": // rotate the top n elements of the stack by m
							// n
							if math.IsNaN(e.scratch[indexOfFirstArg].(float64)) || math.IsInf(e.scratch[indexOfFirstArg].(float64), 1) || math.IsInf(e.scratch[indexOfFirstArg].(float64), -1) || e.scratch[indexOfFirstArg].(float64) <= 0 {
								return newErrSyntax("%s operator requires positive finite integer: %v", token, e.scratch[indexOfFirstArg])
							}
							n := int(e.scratch[indexOfFirstArg].(float64))
							if n > e.scratchHead-1 {
								return newErrSyntax("%s %d items, but only %d on stack", token, n, e.scratchHead-1)
							}
							// m
							if math.IsNaN(e.scratch[indexOfFirstArg+1].(float64)) || math.IsInf(e.scratch[indexOfFirstArg+1].(float64), 1) || math.IsInf(e.scratch[indexOfFirstArg+1].(float64), -1) {
								return newErrSyntax("%s operator requires positive finite integer: %v", token, e.scratch[indexOfFirstArg+1])
							}
							m := int(e.scratch[indexOfFirstArg+1].(float64))
							if m > e.scratchHead-1 {
								return newErrSyntax("%s %d items, but only %d on stack", token, m, e.scratchHead-1)
							}
							// cannot roll if any are operators
							for argIdx = indexOfFirstArg - n; argIdx < indexOfFirstArg; argIdx++ {
								if !e.isFloat[argIdx] {
									if _, ok = arity[e.scratch[argIdx].(string)]; ok {
										cannotSimplify = true
										break
									}
								}
							}
							if !cannotSimplify {
								var items []interface{}
								// TODO: optimize this
								for j := 0; j < 3; j++ {
									for i := 0; i < n; i++ {
										items = append(items, e.scratch[i+indexOfFirstArg-n])
									}
								}
								first := len(items)/3 - m
								last := first + n
								copy(e.scratch[indexOfFirstArg-n:], items[first:last])
								e.scratchHead -= 2 // drop the count
								stackUpdated = true
							}
						case "SORT":
							if math.IsNaN(e.scratch[indexOfFirstArg].(float64)) || math.IsInf(e.scratch[indexOfFirstArg].(float64), 1) || math.IsInf(e.scratch[indexOfFirstArg].(float64), -1) || e.scratch[indexOfFirstArg].(float64) <= 0 {
								return newErrSyntax("%s operator requires positive finite integer: %v", token, e.scratch[indexOfFirstArg])
							}
							count = int(e.scratch[indexOfFirstArg].(float64))
							if count > e.scratchHead-1 {
								return newErrSyntax("%s %d items, but only %d on stack", token, count, e.scratchHead-1)
							}
							items := make([]float64, count)
							for argIdx = indexOfFirstArg - count; argIdx < indexOfFirstArg; argIdx++ {
								if !e.isFloat[argIdx] {
									cannotSimplify = true
									break
								}
								items[argIdx+indexOfFirstArg-count] = e.scratch[argIdx].(float64)
							}
							if !cannotSimplify {
								sort.Float64s(items)
								for argIdx = indexOfFirstArg - count; argIdx < indexOfFirstArg; argIdx++ {
									e.scratch[argIdx] = items[argIdx+indexOfFirstArg-count]
									e.isFloat[argIdx] = true
								}
								e.scratchHead-- // drop the count
								stackUpdated = true
							}
						}
					}

					if cannotSimplify {
						e.scratch[e.scratchHead] = token
						e.isFloat[e.scratchHead] = false
						e.scratchHead++
					} else if !stackUpdated {
						_, isFloat = result.(float64)
						e.scratchHead -= opArity.popCount
						e.scratch[e.scratchHead] = result
						e.isFloat[e.scratchHead] = isFloat
						e.scratchHead++
					}
				} else if value, err = strconv.ParseFloat(token, 64); err == nil {
					// token is the string representation of a number
					e.scratch[e.scratchHead] = value
					e.isFloat[e.scratchHead] = true
					e.scratchHead++
				} else if value, ok = bindings[token]; ok {
					// token is a symbol
					e.scratch[e.scratchHead] = value
					e.isFloat[e.scratchHead] = true
					e.scratchHead++
					// } else if _, ok = series[token]; ok {
					// 	// token is a label for a series
					// 	e.scratch[e.head] = token
					// 	e.isFloat[e.head] = false
					// 	e.head++
				} else {
					// cannot resolve token with the current bindings
					if e.openVariables == nil {
						e.openVariables = make([]string, 0)
					}
					e.openVariables = append(e.openVariables, token)
					e.scratch[e.scratchHead] = token
					e.isFloat[e.scratchHead] = false
					e.scratchHead++
				}
			}
		default:
			return newErrSyntax("unexpected token type at position %d: %v", tokIdx+1, tok)
		}
	}
	return nil
}
