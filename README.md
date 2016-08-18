# gorpn

RPN expression evaluator for Go

## References

http://oss.oetiker.ch/rrdtool/doc/rrdgraph_rpn.en.html
http://linux.die.net/man/1/cdeftutorial

## Usage Examples

### Simple

For RPN expressions that contain all the required data to calculate the result, simply create an
expression and evaluate it.

```Go
    expression, err := gorpn.New("60,24.*")
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
    expression, err := gorpn.New("12,age.*")
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
 * AVG (pop count of items, then compute avg, ignoring all UNK)
 * CEIL
 * FLOOR
 * MEDIAN
 * SMIN: a,b,c,3,SMIN -> min(a,b,c)
 * SMAX: a,b,c,3,SMAX -> max(a,b,c)

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

### Exponentiation and Logarithmic Functions

 * EXP
 * LOG
 * SQRT

### Geometric Functions

 * ATAN (output in radians)
 * ATAN2 (output in radians)
 * COS (input in radians)
 * SIN (input in radians)
 * DEG2RAD
 * RAD2DEG

### Other Supported Constants and Functions

 * DAY (number of seconds in a day)
 * DUP (duplicate value on top of stack)
 * EXC (exchange top two items on stack)
 * HOUR (number of seconds in an hour)
 * INF (push +Inf on stack)
 * LIMIT (pop 2 and define inclusive range. pop third. if third in range, push it back, otherwise push UNK. if any of 3 numbers is UNK or ±Inf, push UNK)
 * MAX (UNK if either number is UNK)
 * MIN (UNK if either number is UNK)
 * MINUTE (number of seconds in a minute)
 * NEGINF (push -Inf on stack)
 * NOW (push number of seconds since epoch)
 * POP (discard top element of stack)
 * REV (pop count of items. then pop that many items. reverse, then push back)
 * SORT (pop count of items. then pop that many items. sort, then push back)
 * STEPWIDTH (current step measured in seconds)
 * TREND (create a "sliding window" average of another data series)
 * TRENDNAN (create a "sliding window" average of another data series)
 * UNKN (push UNK)
 * WEEK (number of seconds in a week)

### Features Supported with Variable Binding

The following features are supported, however they only make sense while evaluating in the context
of a set of bindings. See below for more information.

 * COUNT
 * LTIME
 * NEWDAY (push 1 if datum is first datum for day)
 * NEWMONTH (push 1 if datum is first datum for month)
 * NEWWEEK (push 1 if datum is first datum for week)
 * NEWYEAR (push 1 if datum is first datum for year)
 * TIME

## Unsupported Features

The following features have yet to be implemented in this library.

 * PREDICT
 * PREDICTSIGMA
 * PREV
 * PREV(vname)

 * STDEV: a,b,c,3,STDEV -> stdev(a,b,c)
 * POW: a,b,POW -> a**b
 * PERCENT: a,b,c,95,3,PERCENT -> find 95percentile of a,b,c

## Variable Binding

It is useful to send an RPN expression, along with a map of variable names to their respective
values, to the Evaluate method to calculate the expression value in the context of the provided
bindings.

Recall the simple example above, where the RPN expression `60,24,*` does not need any bindings
because all the information required to calculate the result is in the expression. Likewise, the
expression `1,2,3,4,5,6,7,8,9,10,AVG` contains all the information needed to calculate the result.

```Go
    expression, err := gorpn.New("60,24.*")
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
