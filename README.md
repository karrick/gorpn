# gorpn

RPN expression evaluator for Go

# Reference

http://oss.oetiker.ch/rrdtool/doc/rrdgraph_rpn.en.html

# RPN Expression

RPN expression := vname | operator | value [ , RPN expression ]

CDEF instructions works on each data point in the graph.
VDEF instructions work on entire data set in one run. (VDEF instructions only support a limited list of functions.)

# Supported Functions

 * +, -, *, /, %
 * ABS
 * ADDNAN (add, but if one num is NaN/UNK, treat it as zero. if both NaN/UNK, then return NaN/UNK)
 * ATAN, ATAN2 (output in radians)
 * AVG (pop count of items, then compute avg, ignoring all UNK)
 * COUNT (pushes 1 if this is the first value of the data set, the number 2 if it's the second, and so on)
 * DEG2RAD
 * DUP (duplicate value on top of stack)
 * EXC (exchange top two items on stack)
 * FLOOR, CEIL
 * IF (treats 0, UNK, and ±Inf as false)
 * INF (push +Inf on stack)
 * ISINF (is top ±Inf ? push 1 for true or 0 for false)
 * LIMIT (pop 2 and define inclusive range. pop third. if third in range, push it back, otherwise push UNK. if any of 3 numbers is UNK or ±Inf, push UNK)
 * LT, LE, GT, GE, EQ, NE (push 1 for true, or 0 for false)
 * MAX (UNK if either number is UNK)
 * MIN (UNK if either number is UNK)
 * NEGINF (push -Inf on stack)
 * NOW (push number of seconds since epoch)
 * POP (discard top element of stack)
 * RAD2DEG
 * REV (pop count of items. then pop that many items. reverse, then push back)
 * SIN, COS, LOG, EXP, SQRT
 * SORT (pop count of items. then pop that many items. sort, then push back)
 * TREND, TRENDNAN (create a "sliding window" average of another data series)
 * UN (is top UNK ? push 1 for true or 0 for false)
 * UNKN (push UNK)

# Unsupported Functions

 * TIME, LTIME
 * PREDICT, PREDICTSIGMA
 * PREV, PREV(vname)

# Implementation Notes

 * UNKN implemented as NaN.
 * COUNT implemented with special binding '_count_' passed to evaluate. This prevents using this as a label.
