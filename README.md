# gorpn

RPN expression evaluator for Go

# Reference

http://oss.oetiker.ch/rrdtool/doc/rrdgraph_rpn.en.html

# RPN Expression

RPN expression := vname | operator | value [ , RPN expression ]

CDEF instructions works on each data point in the graph.
VDEF instructions work on entire data set in one run. (VDEF instructions only support a limited list of functions.)

# Supported Functions

## Algebraic Functions

 * +, -, *, /, %
 * ABS
 * ADDNAN (add, but if one num is NaN/UNK, treat it as zero. if both NaN/UNK, then return NaN/UNK)
 * AVG (pop count of items, then compute avg, ignoring all UNK)
 * CEIL
 * FLOOR

## Boolean Functions

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

## Geometric Functions

 * ATAN (output in radians)
 * ATAN2 (output in radians)
 * COS (input in radians)
 * SIN (input in radians)
 * DEG2RAD
 * RAD2DEG

## Exponentiation and Logarithms

 * EXP
 * LOG
 * SQRT

## Other Supported Functions

 * DUP (duplicate value on top of stack)
 * EXC (exchange top two items on stack)
 * LIMIT (pop 2 and define inclusive range. pop third. if third in range, push it back, otherwise push UNK. if any of 3 numbers is UNK or ±Inf, push UNK)
 * MAX (UNK if either number is UNK)
 * MIN (UNK if either number is UNK)
 * POP (discard top element of stack)
 * REV (pop count of items. then pop that many items. reverse, then push back)
 * SORT (pop count of items. then pop that many items. sort, then push back)

# Unsupported Functions

 * COUNT (pushes 1 if this is the first value of the data set, the number 2 if it's the second, and so on)
 * LTIME
 * PREDICT
 * PREDICTSIGMA
 * PREV
 * PREV(vname)
 * TIME
 * TREND (create a "sliding window" average of another data series)
 * TRENDNAN (create a "sliding window" average of another data series)

# Supported Constants

 * DAY (number of seconds in a day)
 * HOUR (number of seconds in an hour)
 * INF (push +Inf on stack)
 * MINUTE (number of seconds in a minute)
 * NEGINF (push -Inf on stack)
 * NOW (push number of seconds since epoch)
 * UNKN (push UNK)
 * WEEK (number of seconds in a week)

# Implementation Notes

 * UNKN implemented as NaN.
