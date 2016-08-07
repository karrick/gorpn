package gorpn

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultDelimiter          = ","
	DefaultSecondsPerInterval = 300
)

// type arityTuple [3]int
type arityTuple struct {
	popCount, floatOffset, floatCount, nonOperatorOffset, nonOperatorCount int
}

// arity resolves to the number of items an operation must pop, and
// how many of those must be floats
var arity = map[string]arityTuple{
	"%":        {2, 2, 2, 0, 0},
	"*":        {2, 2, 2, 0, 0},
	"+":        {2, 2, 2, 0, 0},
	"-":        {2, 2, 2, 0, 0},
	"/":        {2, 2, 2, 0, 0},
	"ABS":      {1, 1, 1, 0, 0},
	"ADDNAN":   {2, 2, 2, 0, 0},
	"ATAN":     {1, 1, 1, 0, 0},
	"ATAN2":    {2, 2, 2, 0, 0},
	"AVG":      {1, 1, 1, 0, 0}, // other operands must be floats
	"CEIL":     {1, 1, 1, 0, 0},
	"COPY":     {1, 1, 1, 0, 0}, // other operands cannot be operators
	"COS":      {1, 1, 1, 0, 0},
	"DEG2RAD":  {1, 1, 1, 0, 0},
	"DEPTH":    {0, 0, 0, 0, 0},
	"DUP":      {1, 0, 0, 1, 1}, // equivalent to: 1,COPY
	"EQ":       {2, 0, 0, 2, 2},
	"EXC":      {2, 0, 0, 2, 2}, // equivalent to: 2,REV
	"EXP":      {1, 1, 1, 0, 0},
	"FLOOR":    {1, 1, 1, 0, 0},
	"GE":       {2, 0, 0, 2, 2},
	"GT":       {2, 0, 0, 2, 2},
	"IF":       {3, 3, 1, 2, 2}, // a,b,c,IF
	"INDEX":    {1, 1, 1, 0, 0}, // other operands cannot be operators
	"ISINF":    {1, 1, 1, 0, 0},
	"LE":       {2, 0, 0, 2, 2},
	"LIMIT":    {3, 3, 3, 0, 0},
	"LOG":      {1, 1, 1, 0, 0},
	"LT":       {2, 0, 0, 2, 2},
	"MAX":      {2, 0, 0, 2, 2},
	"MAXNAN":   {2, 0, 0, 2, 2},
	"MEDIAN":   {1, 1, 1, 0, 0}, // other operands must be floats
	"MIN":      {2, 0, 0, 2, 2},
	"MINNAN":   {2, 0, 0, 2, 2},
	"NE":       {2, 0, 0, 2, 2},
	"POP":      {1, 0, 0, 0, 0},
	"RAD2DEG":  {1, 1, 1, 0, 0},
	"REV":      {1, 1, 1, 0, 0}, // other operands cannot be operators
	"ROLL":     {2, 2, 2, 0, 0}, // n,m,ROLL (rotate the top n elements of the stack by m)
	"SIN":      {1, 1, 1, 0, 0},
	"SORT":     {1, 1, 1, 0, 0}, // other operands must be floats
	"SQRT":     {1, 1, 1, 0, 0},
	"TREND":    {2, 1, 1, 2, 1}, // label,count,TREND
	"TRENDNAN": {2, 1, 1, 2, 1}, // label,count,TRENDNAN
	"UN":       {1, 1, 1, 0, 0},
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

// ErrBadBindingType error is returned when one or more bindings have
// a type that is neither a float64 nor a slice of float64 values.
type ErrBadBindingType struct {
	t string
}

// Error returns the error string representation for ErrBadBindingType
// errors.
func (e ErrBadBindingType) Error() string {
	return "bad binding type for " + string(e.t)
}

// ErrOpenBindings error is returned when one or more open bindings
// remain when evaluating a RPN Expression.
type ErrOpenBindings []string

// Error returns the error string representation for ErrOpenVariables
// errors.
func (e ErrOpenBindings) Error() string {
	return "open bindings: " + strings.Join(e, ",")
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
	delimiter                string
	openBindings             map[string]int // count of number of instances
	secondsPerInterval       float64
	tokens                   []interface{} // components of the expression
	performTimeSubstitutions bool
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
	e := &Expression{
		delimiter:          DefaultDelimiter,
		secondsPerInterval: DefaultSecondsPerInterval,
	}
	for _, setter := range setters {
		if err := setter(e); err != nil {
			return nil, err
		}
	}
	tokens := strings.Split(someExpression, e.delimiter)
	e.scratchSize = len(tokens)

	e.tokens = make([]interface{}, e.scratchSize)
	for idx, token := range tokens {
		switch token {
		case "NOW", "TIME", "LTIME", "NEWDAY", "NEWWEEK", "NEWMONTH", "NEWYEAR":
			e.performTimeSubstitutions = true
		case "DUP":
			e.scratchSize++
		}
		e.tokens[idx] = token
	}
	// scratchSize may be larger than it was before above loop
	e.scratch = make([]interface{}, e.scratchSize)
	e.isFloat = make([]bool, e.scratchSize)
	return e.Partial(nil)
}

// ExpressionSetter represents a function that modifies an RPN
// Expression.
type ExpressionSetter func(*Expression) error

// Delimiter allows changing the expected delimiter for an RPN
// Expression from the default delimiter, the comma.
func Delimiter(someByte string) ExpressionSetter {
	return func(e *Expression) error {
		e.delimiter = someByte
		return nil
	}
}

// SecondsPerInterval allows changing the expected number of seconds per interval to be used when
// evaluating an RPN Expression from the default value of 300..
func SecondsPerInterval(seconds float64) ExpressionSetter {
	return func(e *Expression) error {
		e.secondsPerInterval = seconds
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
	return strings.Join(strs, e.delimiter)
}

// Partial creates a new Expression by partial application of the
// parameter bindings. With the additional bindings, it attempts to
// further simplify the expression.
func (e *Expression) Partial(bindings map[string]interface{}) (*Expression, error) {
	// NOTE: We leave exp.performTimeSubstitutions as its default boolean value of false,
	// preventing time substitutions from being made during this simplify operation
	exp := &Expression{
		delimiter:          e.delimiter,
		secondsPerInterval: e.secondsPerInterval,
		tokens:             make([]interface{}, len(e.tokens)),
		scratchSize:        e.scratchSize,
		scratch:            make([]interface{}, e.scratchSize),
		isFloat:            make([]bool, e.scratchSize),
	}
	copy(exp.tokens, e.tokens)

	if err := exp.simplify(bindings); err != nil {
		return nil, err
	}

	// exp will need to know about time when Evaluate is called on it
	exp.performTimeSubstitutions = e.performTimeSubstitutions

	// promote what's remaining in work area to new simplified stored program
	exp.tokens = exp.tokens[:exp.scratchHead] // first, shrink tokens slice
	copy(exp.tokens, exp.scratch)             // then copy

	return exp, nil
}

// Evaluate evaluates the Expression after applying the parameter
// bindings.
func (e *Expression) Evaluate(bindings map[string]interface{}) (float64, error) {
	var err error

	if err = e.simplify(bindings); err != nil {
		return 0, err
	}

	var openBindings []string
	for k, v := range e.openBindings {
		if v > 0 {
			switch k {
			case "LTIME", "NEWDAY", "NEWWEEK", "NEWMONTH", "NEWYEAR":
				// NOTE: these tokens actually require TIME to be bound
				openBindings = append(openBindings, "TIME")
			default:
				openBindings = append(openBindings, k)
			}
		}
	}
	if len(openBindings) > 0 {
		return 0, ErrOpenBindings(openBindings)
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
	return e.valid(nil)
}

func (e Expression) valid(bindings map[string]interface{}) bool {
	err := e.simplify(bindings)
	if err != nil {
		return false
	}
	if len(e.openBindings) > 0 {
		for k := range e.openBindings {
			bindings[k] = 0 // FIXME: some bindings will need series rather than number
		}
		return e.valid(bindings)
	}
	if e.scratchHead != 1 {
		return false
	}
	return e.isFloat[0]
}

func julietTime(secondsSinceEpoch int) (time.Time, int64) {
	julietTime := time.Unix(int64(secondsSinceEpoch), 0) // Juliet time zone is "local" time zone
	_, julietOffset := julietTime.Zone()
	julietSeconds := julietTime.Unix() + int64(julietOffset)
	// fmt.Printf("julietTime: %v; julietSeconds: %v\n", julietTime, julietSeconds)
	// fmt.Printf("zuluTime: %v; zuluSeconds: %v\n", julietTime.UTC(), julietTime.UTC().Unix())
	return julietTime, julietSeconds
}

func (e *Expression) simplify(bindings map[string]interface{}) error {
	// NOTE: scratch is not local variable so Partial has access to it
	// TODO: change method signature to pass it back and make it local

	var err error

	bindings, err = coerceMapValuesToFloat64(bindings)
	if err != nil {
		return err
	}

	// with a fresh start comes fresh workspace
	e.scratchHead = 0
	e.openBindings = make(map[string]int)

	// heisenberg principle, realized: it takes time to observe the time, so do it only once
	var isNowSet, isTimeSet bool
	var now interface{} = "NOW"
	var zSeconds interface{} = "TIME"
	var jSeconds interface{} = "LTIME"
	var jTime time.Time

	if e.performTimeSubstitutions {
		now = float64(time.Now().Unix())
		isNowSet = true

		if tm, ok := bindings["TIME"]; ok {
			if secondsSinceEpoch, ok := tm.(float64); ok {
				jTime, jSeconds = julietTime(int(secondsSinceEpoch))
				jSeconds = float64(jSeconds.(int64))
				zSeconds = float64(jTime.Unix())
				isTimeSet = true
			}
		}
	}

	// variables outside of loop to reduce allocations
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
			case "NEWDAY", "NEWWEEK", "NEWMONTH", "NEWYEAR":
				if isTimeSet {
					var n float64
					t := time.Unix(int64(zSeconds.(float64)), 0)
					switch token {
					case "NEWDAY":
						if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 {
							n = 1
						}
					case "NEWWEEK":
						if t.Weekday() == time.Sunday {
							n = 1
						}
					case "NEWMONTH":
						if t.Day() == 1 {
							n = 1
						}
					case "NEWYEAR":
						if t.Month() == time.January && t.Day() == 1 {
							n = 1
						}
					}
					e.scratch[e.scratchHead] = n
				} else {
					e.scratch[e.scratchHead] = token
					e.openBindings[token] = e.openBindings[token] + 1
				}
				e.isFloat[e.scratchHead] = isTimeSet
				e.scratchHead++
			case "NOW":
				e.scratch[e.scratchHead] = now
				e.isFloat[e.scratchHead] = isNowSet
				if !isNowSet {
					e.openBindings[token] = e.openBindings[token] + 1
				}
				e.scratchHead++
			case "LTIME":
				e.scratch[e.scratchHead] = jSeconds
				e.isFloat[e.scratchHead] = isTimeSet
				if !isTimeSet {
					e.openBindings[token] = e.openBindings[token] + 1
				}
				e.scratchHead++
			case "STEPWIDTH":
				e.scratch[e.scratchHead] = e.secondsPerInterval
				e.isFloat[e.scratchHead] = true
				e.scratchHead++
			case "TIME":
				e.scratch[e.scratchHead] = zSeconds
				e.isFloat[e.scratchHead] = isTimeSet
				if !isTimeSet {
					e.openBindings[token] = e.openBindings[token] + 1
				}
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

					// fmt.Println("FLOAT CHECK: e.tokens:", e.tokens, "e.scratch:", e.scratch[:e.scratchHead], "opArity:", opArity, "floatOffset:", opArity.floatOffset, "floatCount:", opArity.floatCount)
					for argIdx = e.scratchHead - opArity.floatOffset; argIdx < e.scratchHead-opArity.floatOffset+opArity.floatCount; argIdx++ {
						// fmt.Printf("argIndex: %d; scratch: %v\n", argIdx, e.scratch[argIdx])
						if _, isFloat = e.scratch[argIdx].(float64); !isFloat {
							// fmt.Println("found non float:", e.scratch[argIdx])
							cannotSimplify = true
							break
						}
					}

					// fmt.Println("NOT OPERATOR CHECK: e.tokens:", e.tokens, "e.scratch:", e.scratch[:e.scratchHead], "opArity.nonOperatorOffset:", opArity.nonOperatorOffset, "opArity.nonOperatorCount:", opArity.nonOperatorCount)
					for argIdx = e.scratchHead - opArity.nonOperatorOffset; argIdx < e.scratchHead-opArity.nonOperatorOffset+opArity.nonOperatorCount; argIdx++ {
						// fmt.Printf("argIndex: %d; scratch: %v\n", argIdx, e.scratch[argIdx])
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
						case "ROLL": // n,m,ROLL -- rotate the top n elements of the stack by m
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
						case "TREND": // label,count,TREND
							// get the count
							v := e.scratch[indexOfFirstArg+1].(float64)
							if math.IsNaN(v) || math.IsInf(v, 1) || math.IsInf(v, -1) || v <= 0 {
								return newErrSyntax("%s operator requires positive finite integer: %v", token, v)
							}
							count = int(math.Ceil(v / float64(e.secondsPerInterval)))
							// get series label
							label, ok := e.scratch[indexOfFirstArg].(string)
							if !ok {
								return newErrSyntax("%s operator requires label but found %T: %v", token, e.scratch[indexOfFirstArg], e.scratch[indexOfFirstArg])
							}
							// log.Printf("label: %q\n", label)
							series, ok := bindings[label]
							if !ok {
								// log.Printf("cannot find label binding: %q", label)
								cannotSimplify = true
							} else {
								if s, ok := series.([]float64); ok {
									// log.Printf("label bound to []float64")
									if count > len(s) {
										return newErrSyntax("%s operand specifies %d values, but only %d available", token, count, len(s))
									} else {
										e.openBindings[label] = e.openBindings[label] - 1
										total = 0
										used = 0
										for argIdx = len(s) - count; argIdx < len(s); argIdx++ {
											total += s[argIdx]
											used++
										}
										e.scratchHead -= opArity.popCount
										e.scratch[e.scratchHead] = total / float64(used)
										e.isFloat[e.scratchHead] = true
										e.scratchHead++
										stackUpdated = true
									}
								} else {
									return newErrSyntax("%s operand specifies %q label, which is not a series of numbers: %T", token, label, s)
								}
							}
						case "TRENDNAN": // label,count,TRENDNAN
							// get the count
							v := e.scratch[indexOfFirstArg+1].(float64)
							if math.IsNaN(v) || math.IsInf(v, 1) || math.IsInf(v, -1) || v <= 0 {
								return newErrSyntax("%s operator requires positive finite integer: %v", token, v)
							}
							count = int(math.Ceil(v / e.secondsPerInterval))
							// get series label
							label, ok := e.scratch[indexOfFirstArg].(string)
							if !ok {
								return newErrSyntax("%s operator requires label but found %T: %v", token, e.scratch[indexOfFirstArg], e.scratch[indexOfFirstArg])
							}
							// log.Printf("label: %q\n", label)
							series, ok := bindings[label]
							if !ok {
								// log.Printf("cannot find label binding: %q", label)
								cannotSimplify = true
							} else {
								if s, ok := series.([]float64); ok {
									// log.Printf("label bound to []float64")
									if count > len(s) {
										return newErrSyntax("%s operand specifies %d values, but only %d available", token, count, len(s))
									} else {
										e.openBindings[label] = e.openBindings[label] - 1
										total = 0
										used = 0
										for argIdx = len(s) - count; argIdx < len(s); argIdx++ {
											if !math.IsNaN(s[argIdx]) {
												total += s[argIdx]
												used++
											}
										}
										e.scratchHead -= opArity.popCount
										e.scratch[e.scratchHead] = total / float64(used)
										e.isFloat[e.scratchHead] = true
										e.scratchHead++
										stackUpdated = true
									}
								} else {
									return newErrSyntax("%s operand specifies %q label, which is not a series of numbers: %T", token, label, s)
								}
							}
						case "MEDIAN":
							if math.IsNaN(e.scratch[indexOfFirstArg].(float64)) || math.IsInf(e.scratch[indexOfFirstArg].(float64), 1) || math.IsInf(e.scratch[indexOfFirstArg].(float64), -1) || e.scratch[indexOfFirstArg].(float64) <= 0 {
								return newErrSyntax("%s operator requires positive finite integer: %v", token, e.scratch[indexOfFirstArg])
							}
							count = int(e.scratch[indexOfFirstArg].(float64))
							if count > e.scratchHead-1 {
								return newErrSyntax("%s %d items, but only %d on stack", token, count, e.scratchHead-1)
							}
							if count == 1 {
								// pin-hole optimization for 1 item
								e.scratchHead -= 1
								e.scratch[e.scratchHead] = e.scratch[argIdx]
								_, e.isFloat[e.scratchHead] = e.scratch[argIdx].(float64)
								stackUpdated = true
							} else {
								items := make([]float64, 0, count)
								for argIdx = indexOfFirstArg - count; argIdx < indexOfFirstArg; argIdx++ {
									if !e.isFloat[argIdx] {
										cannotSimplify = true
										break
									}
									items = append(items, e.scratch[argIdx].(float64))
								}
								if !cannotSimplify {
									sort.Float64s(items)
									e.scratchHead -= 1 + count
									middle := count / 2
									// even or odd?
									if count%2 == 0 {
										e.scratch[e.scratchHead] = (items[middle-1] + items[middle]) / 2
									} else {
										e.scratch[e.scratchHead] = items[middle]
									}
									e.isFloat[e.scratchHead] = true
									e.scratchHead++
									stackUpdated = true
								}
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
				} else if val, ok := bindings[token]; ok {
					// token is a symbol to a binding
					switch v := val.(type) {
					case float64:
						// token is a symbol that binds to a variable
						e.scratch[e.scratchHead] = v
						e.isFloat[e.scratchHead] = true
						e.scratchHead++
					case []float64:
						// token is a symbol that binds to a series
						e.openBindings[token] = e.openBindings[token] + 1
						e.scratch[e.scratchHead] = token
						e.isFloat[e.scratchHead] = false
						e.scratchHead++
					}
				} else {
					// cannot resolve token with the current bindings
					e.openBindings[token] = e.openBindings[token] + 1
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

func coerceMapValuesToFloat64(bindings map[string]interface{}) (map[string]interface{}, error) {
	var err error
	newBindings := make(map[string]interface{})

	for key, value := range bindings {
		switch reflect.TypeOf(value).Kind() {
		case reflect.Slice:
			newBindings[key], err = coerceValuesToFloat64(value)
			if err != nil {
				return nil, ErrBadBindingType{fmt.Sprintf("%q: %q", key, err.(ErrBadBindingType).t)}
			}
		default:
			newBindings[key], err = coerceValueToFloat64(value)
			if err != nil {
				return nil, ErrBadBindingType{fmt.Sprintf("%q: %q", key, err.(ErrBadBindingType).t)}
			}
		}
	}

	return newBindings, nil
}

func coerceValuesToFloat64(value interface{}) ([]float64, error) {
	var newList []float64

	switch oldList := value.(type) {
	case []float64:
		// already have what we need
		return oldList, nil
	case []int:
		for _, v := range oldList {
			newList = append(newList, float64(v))
		}
	case []interface{}:
		// slice of unknowns: need to coerce each one dynamically
		for _, v := range oldList {
			cf, err := coerceValueToFloat64(v)
			if err != nil {
				return nil, ErrBadBindingType{fmt.Sprintf("%T", v)}
			}
			newList = append(newList, cf)
		}
	case []float32:
		for _, v := range oldList {
			newList = append(newList, float64(v))
		}
	case []int32:
		for _, v := range oldList {
			newList = append(newList, float64(v))
		}
	case []int64:
		for _, v := range oldList {
			newList = append(newList, float64(v))
		}
	default:
		return nil, ErrBadBindingType{fmt.Sprintf("%T", oldList)}
	}

	return newList, nil
}

func coerceValueToFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	default:
		return 0, ErrBadBindingType{fmt.Sprintf("%T", v)}
	}
}
