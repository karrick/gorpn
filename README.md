# gorpn

RPN expression evaluator for Go

## References

- http://oss.oetiker.ch/rrdtool/doc/rrdgraph_rpn.en.html

- http://linux.die.net/man/1/cdeftutorial

## Usage Examples

### Simple

For RPN expressions that contain all the required data to calculate the result, simply create an
expression and evaluate it.

```Go
    expression, err := gorpn.New("60,24,*")
    if err != nil {
        panic(err)
    }
    result, err := expression.Evaluate(nil)
    if err != nil {
        panic(err)
    }
```

### With Variable Bindings

For RPN expressions that do _not_ contain all the required data, and require variable bindings,
provide them in the form of a map of string variable names to their respective numerical values.

```Go
    expression, err := gorpn.New("12,age,*")
    if err != nil {
        panic(err)
    }
    bindings := map[string]interface{} {
        "age": 21,
    }
    result, err := expression.Evaluate(bindings)
    if err != nil {
        panic(err)
    }
```

## Supported Features

### Algebraic Functions

 * +, -, *, /, %
 * ABS
 * ADDNAN (add, but if one num is NaN/UNK, treat it as zero. if both NaN/UNK, then return NaN/UNK)
 * ATAN (output in radians)
 * ATAN2 (output in radians)
 * CEIL
 * COS (input in radians)
 * DEG2RAD
 * EXP: a,EXP -> a^_e_, where _e_ is the natural number
 * FLOOR
 * LOG: a,LOG -> log base _e_ of a, where _e_ is the natural number
 * POW: a,b,POW -> a^b
 * RAD2DEG
 * SIN (input in radians)
 * SMAX: a,b,c,3,SMAX -> max(a,b,c)
 * SMIN: a,b,c,3,SMIN -> min(a,b,c)
 * SQRT

### Boolean Functions

Each logical function pushes 1 for 0, and 0 for false.

 * EQ (=)
 * GE (>=)
 * GT (>)
 * IF (treats 0, UNK, and ±Inf as false)
 * ISINF (is top ±Inf)
 * LE (<=)
 * LT (<)
 * NE (!=)
 * UN (is top of stack UNK?)

### Comparing Values

Pop two elements from the stack and pushes back the larger or smaller
element, depending on the name.  Infinite is larger than every other
number.  If either of the numbers are UNK, then pushes UNK back on the
stack.

 * MAX
 * MIN

These versions work similar to above, with the exception that if
either of the two numbers popped off the stack are UNK, then it pushes
the other number.

 * MAXNAN
 * MINNAN 

### Set Operations

 * count,SORT: Pop count of items, then pop that many items. Sort, then push all items back.
 * count,REV: Pop count of items, then pop that many items. Reverse, then push all items back.
 * count,AVG: Pop count of items, then compute mean, ignoring all UNK. Push mean back.
 * count,MAD: a,b,c,3,MAD -> median absolute deviation of [a, b, c]
 * count,MEDIAN: a,b,c,3,MEDIAN -> median of [a, b, c]
 * percentile,count,PERCENT: a,b,c,95,3,PERCENT -> find 95percentile of a,b,c using the nearest rank method (https://en.wikipedia.org/wiki/Percentile)
 * count,STDEV: a,b,c,3,STDEV -> stdev(a,b,c), ignoring all UNK
 * count,TREND: create a "sliding window" average of another data series
 * count,TRENDNAN: create a "sliding window" average of another data series

### Other Supported Constants and Functions

 * DAY: number of seconds in a day
 * HOUR: number of seconds in an hour
 * INF: push +Inf on stack
 * LIMIT: pop 2 and define inclusive range. pop third. if third in range, push it back, otherwise push UNK. if any of 3 numbers is UNK or ±Inf, push UNK
 * MINUTE: number of seconds in a minute
 * NEGINF: push -Inf on stack
 * NOW: push number of seconds since epoch
 * STEPWIDTH: current step measured in seconds
 * UNKN: push UNK
 * WEEK: number of seconds in a week

### Features Supported with Variable Binding

The following features are supported, however they only make sense while evaluating in the context
of a set of bindings. See below for more information.

 * COUNT
 * LTIME
 * NEWDAY: push 1 if datum is first datum for day
 * NEWMONTH: push 1 if datum is first datum for month
 * NEWWEEK: push 1 if datum is first datum for week
 * NEWYEAR: push 1 if datum is first datum for year
 * TIME

### Stack Manipulation

 * n,COPY: push a copy of the top _n_ elements onto the stack
 * DEPTH: pushes the current depth of the stack onto the stack
 * DUP: duplicate value on top of stack
 * EXC: exchange top two items on stack
 * n,INDEX: push the _nth_ element onto the stack
 * POP: discard top element of stack
 * n,m,ROLL: rotate the top _n_ elements of the stack by _m_

## Unsupported Features

The following features have yet to be implemented in this library.

 * PREDICT
 * PREDICTSIGMA
 * PREV
 * PREV(vname)

## Variable Binding

It is useful to send an RPN expression, along with a map of variable names to their respective
values, to the Evaluate method to calculate the expression value in the context of the provided
bindings.

Recall the simple example above, where the RPN expression `60,24,*` does not need any bindings
because all the information required to calculate the result is in the expression. Likewise, the
expression `1,2,3,4,5,6,7,8,9,10,AVG` contains all the information needed to calculate the result.

```Go
    expression, err := gorpn.New("60,24,*")
    if err != nil {
        panic(err)
    }
    result, err := expression.Evaluate(nil)
    if err != nil {
        panic(err)
    }
```

However, the RPN expression `12,age,/` requires a binding of the term `age` to its numerical value
in order to evaluate the final result.

Variable bindings are supported by providing a map of string names to their numerical values to the
Evaluate method. Recall that when no bindings are needed, the `nil` value may be sent to Evaluate to
find the RPN result. In the example below,

```Go
    type Datum struct {
        when time.Time
        what float64
    }

    type Series []Datum

    func getValueAtTime(when time.Time, series Series) float64 {
        // magic, returns math.NaN() when value not available for time
    }

    func example(start, end time.Time, interval time.Duration, data map[string]Series) {
        for when := start; end.Before(when); when.Add(interval) {
            bindings := make(map[string]interface{})

            for label, series := range data {
                bindings[label] = getValueAtTime(when, series)
            }

            value, err := exp.Evaluate(bindings)
            // handle error...
        }
    }
```

## Features Supported with Variable Binding

### COUNT

When evaluating an RPN expression, the evaluator for a single expression does not know how many
other expressions have been evaluated. The program that is requesting the evaluation must provide
that information in the form of a binding variable.

### LTIME and TIME, contrasted against NOW

The NOW pseudo-variable is _always_ available during evaulation because it's the number of seconds
since the UNIX epoch at the moment of evaluation. In contrast, TIME in RRD parlance, refers to the
time associated with a particular datum. As a program loops through a bunch of time+value datum
tuples, it will need to bind the time to the TIME symbol in the bindings map provided to Evaluate.

The RPN evaluator does not know the time a particular datum was obtained; that must be provided at
the time of evaluation. But once TIME is provided, other pseudo-variables are available for
evaluation, including LTIME, NEWDAY, NEWWEEK, NEWMONTH, and NEWYEAR.

LTIME, like TIME, corresponds to the time associated with a particular datum. It is calculated from
the bound TIME value provided in the bindings to Evaluate.

```Go
    // as before...

    func example(start, end time.Time, interval time.Duration, data map[string]Series) {
        var count int
        for when := start; end.Before(when); when.Add(interval) {
            count++ // according to the RRD spec, count starts at 1 for the first item in the series

            bindings := make(map[string]interface{})
            bindings["COUNT"] = count
            bindings["TIME"] = when.Unix()

            for label, series := range data {
                bindings[label] = getValueAtTime(when, series)
            }

            value, err := exp.Evaluate(bindings)
            ...
        }
    }
```

# Implementation Notes

## UNKN implemented as NaN.

Perhaps this ought to change. I'm not sure. It seems like it is a decent compromise for now. Let me
know if you experience problems using this library because of this assumption. It may change in the
future. If it does, I hope the library API will remain constant.

```Go
    for _, tm := range times {
        bindings := map[string]interface{}{
            "TIME": tm,
        }
        value, err := exp.Evaluate(bindings)
        ...
    }
```

## PREV

Pushes an unknown value if this is the first value of a data set or otherwise the result of this
CDEF at the previous time step. This allows you to do calculations across the data. This function
cannot be used in VDEF instructions.

## PREV(vname)

Requires COUNT binding.

Pushes an unknown value if this is the first value of a data set or otherwise the result of the
vname variable at the previous time step. This allows you to do calculations across the data. This
function cannot be used in VDEF instructions.
