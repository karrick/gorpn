package gorpn

import (
	"errors"
	"fmt"
	"math"
	"testing"
	"time"
)

func TestNewExpressionEmptyString(t *testing.T) {
	_, err := New("")
	switch err.(type) {
	case ErrSyntax:
	default:
		t.Errorf("Actual: %#v; Expected: %#v", err, ErrSyntax{})
	}
}

func TestNewExpressionInvalidSetter(t *testing.T) {
	badSetter := func(_ *Expression) error {
		return errors.New("foo")
	}
	_, err := New("13", badSetter)
	if err == nil || err.Error() != "foo" {
		t.Errorf("Actual: %#v; Expected: %#v", err, "foo")
	}
}

func TestNewExpressionEmptyToken(t *testing.T) {
	_, err := New(",")
	switch err.(type) {
	case ErrSyntax:
	default:
		t.Errorf("Actual: %#v; Expected: %#v; %T", err, nil, err)
	}

	_, err = New("a,")
	switch err.(type) {
	case ErrSyntax:
	default:
		t.Errorf("Actual: %#v; Expected: %#v; %T", err, nil, err)
	}

	_, err = New(",a")
	switch err.(type) {
	case ErrSyntax:
	default:
		t.Errorf("Actual: %#v; Expected: %#v; %T", err, nil, err)
	}
}

func TestNewExpressionStackUnderflow(t *testing.T) {
	_, err := New("4,*")
	switch err.(type) {
	case ErrSyntax:
	default:
		t.Errorf("Actual: %#v; Expected: %#v; %T", err, nil, err)
	}
}

func TestNewExpressionSimplifyConstants(t *testing.T) {
	list := map[string]string{
		"MINUTE": "60",
		"HOUR":   "3600",
		"DAY":    "86400",
		"WEEK":   "604800",
		// NOTE: The following values get turned into NaN, Inf, and -Inf, but must get
		// changed back to UNKN, INF, and NEGINF when printing.
		"UNKN":   "UNKN",
		"INF":    "INF",
		"NEGINF": "NEGINF",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Errorf("Case: %s; Actual: %s; Expected: %v", input, err, nil)
		} else if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestDivisorNaN(t *testing.T) {
	r := 5 / math.NaN()
	if !math.IsNaN(r) {
		t.Errorf("Actual: %#v; Expected: %#v", r, math.NaN)
	}
}

func TestNewExpressionSimplifiesWhatItCan(t *testing.T) {
	list := map[string]string{
		"5,2,+":    "7",
		"5,2,-":    "3",
		"5,2,*":    "10",
		"5,2,/":    "2.5",
		"5,2,%":    "1",
		"5,NaN,/":  "UNKN", // NaN is represented as UNKN (don't like this)
		"5,UNKN,/": "UNKN", // NaN is represented as UNKN (don't like this)

		"x,x,+": "x,x,+",
		"x,x,-": "x,x,-",
		"x,x,*": "x,x,*",
		"x,x,/": "x,x,/",
		"x,x,%": "x,x,%",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionUnresolvedSymbol(t *testing.T) {
	list := map[string]string{
		"5,foo,+":     "5,foo,+",
		"5,3,+,foo,*": "8,foo,*",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionExamples(t *testing.T) {
	list := map[string]string{
		"0,0,GT,qps,0,0,EQ,-2,0,IF,IF": "-2",
		"1,0,GT,qps,-2,IF":             "qps",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionABS(t *testing.T) {
	list := map[string]string{
		"-1,ABS":     "1",
		"0,ABS":      "0",
		"1,ABS":      "1",
		"NEGINF,ABS": "INF",
		"INF,ABS":    "INF",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionADDNAN(t *testing.T) {
	list := map[string]string{
		"1.1,2.5,ADDNAN":   "3.6",
		"UNKN,2.5,ADDNAN":  "2.5",
		"7.6,UNKN,ADDNAN":  "7.6",
		"UNKN,UNKN,ADDNAN": "UNKN",
		"x,x,ADDNAN":       "x,x,ADDNAN",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionAVG(t *testing.T) {
	errors := map[string]string{
		"1,2,3,-1,AVG":     "syntax error : AVG operator requires positive finite integer: -1",
		"1,2,3,0,AVG":      "syntax error : AVG operator requires positive finite integer: 0",
		"1,2,3,4,AVG":      "syntax error : AVG 4 items, but only 3 on stack",
		"1,2,3,INF,AVG":    "syntax error : AVG operator requires positive finite integer: +Inf",
		"1,2,3,NEGINF,AVG": "syntax error : AVG operator requires positive finite integer: -Inf",
	}
	for i, e := range errors {
		if _, err := New(i); err == nil || err.Error() != e {
			t.Errorf("Case: %s; Actual: %s; Expected: %#v", i, err, e)
		}
	}
	list := map[string]string{
		"a,b,c,3,AVG":      "a,b,c,3,AVG", // cannot average variables
		"13,42,2,AVG":      "27.5",
		"42,13,2,AVG":      "27.5",
		"13,a,ISINF,2,AVG": "13,a,ISINF,2,AVG",
		// AVG ignores UNKN values
		"42,UNKN,13,3,AVG": "27.5",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionCEIL(t *testing.T) {
	list := map[string]string{
		"-0.5,CEIL":   "-0",
		"-1.5,CEIL":   "-1",
		"0.5,CEIL":    "1",
		"INF,CEIL":    "INF",
		"NEGINF,CEIL": "NEGINF",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionCOPY(t *testing.T) {
	errors := map[string]string{
		"1,2,3,-1,COPY":     "syntax error : COPY operator requires positive finite integer: -1",
		"1,2,3,0,COPY":      "syntax error : COPY operator requires positive finite integer: 0",
		"1,2,3,4,COPY":      "syntax error : COPY 4 items, but only 3 on stack",
		"1,2,3,INF,COPY":    "syntax error : COPY operator requires positive finite integer: +Inf",
		"1,2,3,NEGINF,COPY": "syntax error : COPY operator requires positive finite integer: -Inf",
	}
	for i, e := range errors {
		if _, err := New(i); err == nil || err.Error() != e {
			t.Errorf("Case: %s; Actual: %s; Expected: %#v", i, err, e)
		}
	}
	list := map[string]string{
		"1,2,3,d,COPY":   "1,2,3,d,COPY",
		"a,b,EQ,2,COPY":  "a,b,EQ,2,COPY",
		"a,b,c,d,2,COPY": "a,b,c,d,c,d",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

// COUNT

func TestEvaluateCOUNTWithoutCOUNT(t *testing.T) {
	exp, err := New("COUNT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = exp.Evaluate(nil)
	if err == nil || err.Error() != "open bindings: COUNT" {
		t.Errorf("Actual: %s; Expected: %#v", err, "open bindings: COUNT")
	}
}

func TestEvaluateCOUNTWithTime(t *testing.T) {
	exp, err := New("COUNT")
	if err != nil {
		t.Fatal(err)
	}
	value, err := exp.Evaluate(map[string]interface{}{
		"COUNT": 666,
	})
	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
	if int(value) != 666 {
		t.Errorf("Actual: %#v; Expected: %#v", int(value), 666)
	}
}

func TestNewExpressionDEPTH(t *testing.T) {
	list := map[string]string{
		"DEPTH":     "0",
		"a,b,DEPTH": "a,b,2",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionDUP(t *testing.T) {
	errors := map[string]string{
		"DUP": "syntax error : not enough parameters: operator DUP requires 1 operands",
	}
	for i, e := range errors {
		if _, err := New(i); err == nil || err.Error() != e {
			t.Errorf("Case: %s; Actual: %s; Expected: %#v", i, err, e)
		}
	}
	list := map[string]string{
		"13,42,DUP": "13,42,42",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionEQ(t *testing.T) {
	list := map[string]string{
		"5,2,EQ":           "0",
		"5,x,EQ":           "5,x,EQ",
		"x,2,EQ":           "x,2,EQ",
		"INF,INF,EQ":       "1",
		"INF,NEGINF,EQ":    "0",
		"NEGINF,NEGINF,EQ": "1",
		"UNKN,UNKN,EQ":     "0",
		"x,x,EQ":           "1",
		"x,y,EQ":           "x,y,EQ",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if actual, want := exp.String(), output; actual != want {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, actual, want)
		}
	}
}

func TestNewExpressionEXC(t *testing.T) {
	errors := map[string]string{
		"EXC": "syntax error : not enough parameters: operator EXC requires 2 operands",
	}
	for i, e := range errors {
		if _, err := New(i); err == nil || err.Error() != e {
			t.Errorf("Case: %s; Actual: %s; Expected: %#v", i, err, e)
		}
	}
	list := map[string]string{
		"13,42,EXC": "42,13",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionFLOOR(t *testing.T) {
	list := map[string]string{
		"-0.5,FLOOR":   "-1",
		"-1.5,FLOOR":   "-2",
		"0.5,FLOOR":    "0",
		"1.5,FLOOR":    "1",
		"INF,FLOOR":    "INF",
		"NEGINF,FLOOR": "NEGINF",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionGE(t *testing.T) {
	list := map[string]string{
		"2,5,GE":           "0",
		"5,2,GE":           "1",
		"5,x,GE":           "5,x,GE",
		"INF,INF,GE":       "1",
		"INF,NEGINF,GE":    "1",
		"NEGINF,INF,GE":    "0",
		"NEGINF,NEGINF,GE": "1",
		"x,2,GE":           "x,2,GE",
		"x,x,GE":           "1",
		"x,y,GE":           "x,y,GE",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if actual, want := exp.String(), output; actual != want {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, actual, want)
		}
	}
}

func TestNewExpressionGT(t *testing.T) {
	list := map[string]string{
		"2,5,GT":           "0",
		"5,2,GT":           "1",
		"5,x,GT":           "5,x,GT",
		"INF,INF,GT":       "0",
		"INF,NEGINF,GT":    "1",
		"NEGINF,INF,GT":    "0",
		"NEGINF,NEGINF,GT": "0",
		"x,2,GT":           "x,2,GT",
		"x,x,GT":           "0",
		"x,y,GT":           "x,y,GT",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if actual, want := exp.String(), output; actual != want {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, actual, want)
		}
	}
}

func TestNewExpressionGeometric(t *testing.T) {
	list := map[string]string{
		"90,DEG2RAD,SIN":                   "1",
		"180,DEG2RAD,COS":                  "-1",
		fmt.Sprintf("%v,RAD2DEG", math.Pi): "180",
		"1,ATAN":    "0.7853981633974483",
		"1,2,ATAN2": "1.1071487177940904",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionIF(t *testing.T) {
	errors := map[string]string{
		"IF":     "syntax error : not enough parameters: operator IF requires 3 operands",
		"0,IF":   "syntax error : not enough parameters: operator IF requires 3 operands",
		"1,0,IF": "syntax error : not enough parameters: operator IF requires 3 operands",
	}
	for i, e := range errors {
		if _, err := New(i); err == nil || err.Error() != e {
			t.Errorf("Case: %s; Actual: %s; Expected: %#v", i, err, e)
		}
	}
	// A,B,C,IF ==> A ? B : C
	list := map[string]string{
		"NEGINF,1,0,IF":   "1",
		"-1,1,0,IF":       "1",
		"0,1,0,IF":        "0",
		"1,1,0,IF":        "1",
		"2,1,0,IF":        "1",
		"INF,1,0,IF":      "1",
		"UNKN,1,0,IF":     "0",
		"0,ab,bc,IF":      "bc",
		"1,ab,bc,IF":      "ab",
		"1,0,EQ,ab,bc,IF": "bc",
		"1,1,EQ,ab,bc,IF": "ab",
		"qps,1,0,IF":      "qps,1,0,IF", // when predicate is a variable
		"1,2,+,4,5,IF":    "4",
		"1,a,3,+,5,IF":    "1,a,3,+,5,IF",
		"7,2,4,+,5,IF":    "6",
		"7,a,4,+,5,IF":    "7,a,4,+,5,IF",
		"a,7,+,3,5,IF":    "a,7,+,3,5,IF",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %s; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionINDEX(t *testing.T) {
	errors := map[string]string{
		"1,2,3,-1,INDEX":     "syntax error : INDEX operator requires positive finite integer: -1",
		"1,2,3,0,INDEX":      "syntax error : INDEX operator requires positive finite integer: 0",
		"1,2,3,4,INDEX":      "syntax error : INDEX 4 items, but only 3 on stack",
		"1,2,3,INF,INDEX":    "syntax error : INDEX operator requires positive finite integer: +Inf",
		"1,2,3,NEGINF,INDEX": "syntax error : INDEX operator requires positive finite integer: -Inf",
	}
	for i, e := range errors {
		if _, err := New(i); err == nil || err.Error() != e {
			t.Errorf("Case: %s; Actual: %s; Expected: %#v", i, err, e)
		}
	}
	list := map[string]string{
		"a,b,c,d,3,INDEX":        "a,b,c,d,b",
		"1,2,3,a,b,EQ,d,3,INDEX": "1,2,3,a,b,EQ,d,3,INDEX",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionIsInf(t *testing.T) {
	list := map[string]string{
		"-1,ISINF":     "0",
		"0,ISINF":      "0",
		"1,ISINF":      "0",
		"INF,ISINF":    "1",
		"NEGINF,ISINF": "1",
		"UNKN,ISINF":   "0",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionLIMIT(t *testing.T) {
	errors := map[string]string{
		"4,LIMIT":   "syntax error : not enough parameters: operator LIMIT requires 3 operands",
		"3,4,LIMIT": "syntax error : not enough parameters: operator LIMIT requires 3 operands",
	}
	for i, e := range errors {
		if _, err := New(i); err == nil || err.Error() != e {
			t.Errorf("Case: %s; Actual: %s; Expected: %#v", i, err, e)
		}
	}
	list := map[string]string{
		"foo,6,5,10,LIMIT,+": "foo,6,+",
		"-5,-5,10,LIMIT":     "-5",
		"-10,-10,-5,LIMIT":   "-10",
		"-10,-5,10,LIMIT":    "UNKN",
		"10,-5,5,LIMIT":      "UNKN",

		"UNKN,0,10,LIMIT":  "UNKN",
		"-5,UNKN,10,LIMIT": "UNKN",
		"-5,0,UNKN,LIMIT":  "UNKN",

		"INF,0,10,LIMIT":  "UNKN",
		"-5,INF,10,LIMIT": "UNKN",
		"-5,0,INF,LIMIT":  "UNKN",

		"NEGINF,0,10,LIMIT":  "UNKN",
		"-5,NEGINF,10,LIMIT": "UNKN",
		"-5,0,NEGINF,LIMIT":  "UNKN",

		"UNKN,INF,NEGINF,LIMIT": "UNKN",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionLE(t *testing.T) {
	list := map[string]string{
		"2,5,LE":           "1",
		"5,2,LE":           "0",
		"5,x,LE":           "5,x,LE",
		"INF,INF,LE":       "1",
		"INF,NEGINF,LE":    "0",
		"NEGINF,INF,LE":    "1",
		"NEGINF,NEGINF,LE": "1",
		"x,2,LE":           "x,2,LE",
		"x,x,LE":           "1",
		"x,y,LE":           "x,y,LE",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if actual, want := exp.String(), output; actual != want {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, actual, want)
		}
	}
}

func TestNewExpressionLT(t *testing.T) {
	list := map[string]string{
		"2,5,LT":           "1",
		"5,2,LT":           "0",
		"5,x,LT":           "5,x,LT",
		"INF,INF,LT":       "0",
		"INF,NEGINF,LT":    "0",
		"NEGINF,INF,LT":    "1",
		"NEGINF,NEGINF,LT": "0",
		"x,2,LT":           "x,2,LT",
		"x,x,LT":           "0",
		"x,y,LT":           "x,y,LT",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if actual, want := exp.String(), output; actual != want {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, actual, want)
		}
	}
}

func TestNewExpressionLogs(t *testing.T) {
	list := map[string]string{
		"-1,SQRT": "UNKN",
		"0,SQRT":  "0",
		"25,SQRT": "5",
	}
	list[fmt.Sprintf("%v,LOG", math.E)] = "1"
	list["1,EXP"] = fmt.Sprintf("%v", math.E)

	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionMAX(t *testing.T) {
	list := map[string]string{
		"3.6,10.2,MAX":          "10.2",
		"10.2,3.6,MAX":          "10.2",
		"a,a,MAX":               "a",
		"1,a,MAX":               "1,a,MAX",
		"a,1,MAX":               "a,1,MAX",
		"i001_{1},i002_{1},MAX": "i001_{1},i002_{1},MAX",
		// if one is UNKN, result is UNKN
		"UNKN,a,MAX":   "UNKN",
		"a,UNKN,MAX":   "UNKN",
		"UNKN,100,MAX": "UNKN",
		"100,UNKN,MAX": "UNKN",
		// INF is larger than anything else
		"-100,INF,MAX": "INF",
		// NEGINF is smaller than anything else
		"-100,NEGINF,MAX": "-100",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionMAXNAN(t *testing.T) {
	list := map[string]string{
		"3.6,10.2,MAXNAN":          "10.2",
		"10.2,3.6,MAXNAN":          "10.2",
		"a,a,MAXNAN":               "a",
		"1,a,MAXNAN":               "1,a,MAXNAN",
		"a,1,MAXNAN":               "a,1,MAXNAN",
		"i001_{1},i002_{1},MAXNAN": "i001_{1},i002_{1},MAXNAN",
		// if one is UNKN, result is the other
		"UNKN,a,MAXNAN":   "a",
		"a,UNKN,MAXNAN":   "a",
		"UNKN,100,MAXNAN": "100",
		"100,UNKN,MAXNAN": "100",
		// INF is larger than anything else
		"-100,INF,MAXNAN": "INF",
		// NEGINF is smaller than anything else
		"-100,NEGINF,MAXNAN": "-100",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionMIN(t *testing.T) {
	list := map[string]string{
		"3.6,10.2,MIN":          "3.6",
		"10.2,3.6,MIN":          "3.6",
		"a,a,MIN":               "a",
		"1,a,MIN":               "1,a,MIN",
		"a,1,MIN":               "a,1,MIN",
		"i001_{1},i002_{1},MIN": "i001_{1},i002_{1},MIN",
		// if one is UNKN, result is UNKN
		"UNKN,a,MIN":   "UNKN",
		"a,UNKN,MIN":   "UNKN",
		"UNKN,100,MIN": "UNKN",
		"100,UNKN,MIN": "UNKN",
		// INF is larger than anything else
		"-100,INF,MIN": "-100",
		// NEGINF is smaller than anything else
		"-100,NEGINF,MIN": "NEGINF",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionMINNAN(t *testing.T) {
	list := map[string]string{
		"3.6,10.2,MINNAN":          "3.6",
		"10.2,3.6,MINNAN":          "3.6",
		"a,a,MINNAN":               "a",
		"1,a,MINNAN":               "1,a,MINNAN",
		"a,1,MINNAN":               "a,1,MINNAN",
		"i001_{1},i002_{1},MINNAN": "i001_{1},i002_{1},MINNAN",
		// if one is UNKN, result is the other
		"UNKN,a,MINNAN":   "a",
		"a,UNKN,MINNAN":   "a",
		"UNKN,100,MINNAN": "100",
		"100,UNKN,MINNAN": "100",
		// INF is larger than anything else
		"-100,INF,MINNAN": "-100",
		// NEGINF is smaller than anything else
		"-100,NEGINF,MINNAN": "NEGINF",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionNE(t *testing.T) {
	list := map[string]string{
		"2,5,NE":           "1",
		"5,2,NE":           "1",
		"5,x,NE":           "5,x,NE",
		"INF,INF,NE":       "0",
		"INF,NEGINF,NE":    "1",
		"NEGINF,INF,NE":    "1",
		"NEGINF,NEGINF,NE": "0",
		"UNKN,UNKN,NE":     "1",
		"x,2,NE":           "x,2,NE",
		"x,x,NE":           "0",
		"x,y,NE":           "x,y,NE",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if actual, want := exp.String(), output; actual != want {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, actual, want)
		}
	}
}

func TestNewExpressionNOWNeverSimplified(t *testing.T) {
	list := map[string]string{
		"1,NOW": "1,NOW",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionPOP(t *testing.T) {
	errors := map[string]string{
		"POP": "syntax error : not enough parameters: operator POP requires 1 operands",
	}
	for i, e := range errors {
		if _, err := New(i); err == nil || err.Error() != e {
			t.Errorf("Case: %s; Actual: %s; Expected: %#v", i, err, e)
		}
	}
	list := map[string]string{
		"13,42,POP": "13",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionREV(t *testing.T) {
	errors := map[string]string{
		"1,2,3,-1,REV":     "syntax error : REV operator requires positive finite integer: -1",
		"1,2,3,0,REV":      "syntax error : REV operator requires positive finite integer: 0",
		"1,2,3,4,REV":      "syntax error : REV 4 items, but only 3 on stack",
		"1,2,3,INF,REV":    "syntax error : REV operator requires positive finite integer: +Inf",
		"1,2,3,NEGINF,REV": "syntax error : REV operator requires positive finite integer: -Inf",
	}
	for i, e := range errors {
		if _, err := New(i); err == nil || err.Error() != e {
			t.Errorf("Case: %s; Actual: %s; Expected: %#v", i, err, e)
		}
	}
	list := map[string]string{
		"a,b,c,3,REV":            "c,b,a",
		"a,b,EQ,2,REV":           "a,b,EQ,2,REV",
		"UNKN,13,42,666,3,REV,-": "UNKN,666,29",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionROLL(t *testing.T) {
	// ??? unknown cases ???
	// "4,3,2.5,1,ROLL": "syntax error : ",
	// "4,3,2,1.5,ROLL": "syntax error : ",

	errors := map[string]string{
		"1,2,0,3,ROLL":      "syntax error : ROLL operator requires positive finite integer: 0",
		"1,2,3,4,ROLL":      "syntax error : ROLL 4 items, but only 3 on stack",
		"1,2,3,INF,ROLL":    "syntax error : ROLL operator requires positive finite integer: +Inf",
		"1,2,3,NEGINF,ROLL": "syntax error : ROLL operator requires positive finite integer: -Inf",
		"1,2,INF,3,ROLL":    "syntax error : ROLL operator requires positive finite integer: +Inf",
		"1,2,NEGINF,3,ROLL": "syntax error : ROLL operator requires positive finite integer: -Inf",
	}
	for i, e := range errors {
		if _, err := New(i); err == nil || err.Error() != e {
			t.Errorf("Case: %s; Actual: %s; Expected: %#v", i, err, e)
		}
	}
	list := map[string]string{
		"4,3,2,0,ROLL":       "4,3",
		"4,3,2,1,ROLL":       "3,4",
		"4,3,2,1,ROLL,/":     "0.75",
		"5,4,3,2,1,ROLL":     "5,3,4",
		"a,b,+,2,1,ROLL":     "a,b,+,2,1,ROLL",
		"a,b,c,d,3,-1,ROLL":  "a,c,d,b",
		"a,b,c,d,3,1,ROLL":   "a,d,b,c",
		"a,b,c,d,e,4,3,ROLL": "a,c,d,e,b",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if actual, want := exp.String(), output; actual != want {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, actual, want)
		}
	}
}

func TestNewExpressionSORT(t *testing.T) {
	errors := map[string]string{
		"1,2,3,-1,SORT":     "syntax error : SORT operator requires positive finite integer: -1",
		"1,2,3,0,SORT":      "syntax error : SORT operator requires positive finite integer: 0",
		"1,2,3,4,SORT":      "syntax error : SORT 4 items, but only 3 on stack",
		"1,2,3,INF,SORT":    "syntax error : SORT operator requires positive finite integer: +Inf",
		"1,2,3,NEGINF,SORT": "syntax error : SORT operator requires positive finite integer: -Inf",
	}
	for i, e := range errors {
		if _, err := New(i); err == nil || err.Error() != e {
			t.Errorf("Case: %s; Actual: %s; Expected: %#v", i, err, e)
		}
	}
	list := map[string]string{
		"a,b,c,3,SORT":      "a,b,c,3,SORT", // cannot sort variables
		"13,42,2,SORT":      "13,42",
		"42,13,2,SORT":      "13,42",
		"13,a,ISINF,2,SORT": "13,a,ISINF,2,SORT",
		"42,13,2,SORT,-":    "-29",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestNewExpressionTREND(t *testing.T) {
	errors := map[string]string{
		"a,0,TREND":  "syntax error : TREND operator requires positive finite integer: 0",
		"a,-1,TREND": "syntax error : TREND operator requires positive finite integer: -1",
	}
	for i, e := range errors {
		if _, err := New(i); err == nil || err.Error() != e {
			t.Errorf("Case: %s; Actual: %s; Expected: %#v", i, err, e)
		}
	}
	list := map[string]string{
		"a,5,TREND": "a,5,TREND",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if actual, want := exp.String(), output; actual != want {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, actual, want)
		}
	}
}

func TestNewExpressionTRENDNAN(t *testing.T) {
	errors := map[string]string{
		"a,0,TRENDNAN":  "syntax error : TRENDNAN operator requires positive finite integer: 0",
		"a,-1,TRENDNAN": "syntax error : TRENDNAN operator requires positive finite integer: -1",
	}
	for i, e := range errors {
		if _, err := New(i); err == nil || err.Error() != e {
			t.Errorf("Case: %s; Actual: %s; Expected: %#v", i, err, e)
		}
	}
	list := map[string]string{
		"a,5,TRENDNAN": "a,5,TRENDNAN",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if actual, want := exp.String(), output; actual != want {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, actual, want)
		}
	}
}

func TestNewExpressionUN(t *testing.T) {
	errors := map[string]string{
		"UN": "syntax error : not enough parameters: operator UN requires 1 operands",
	}
	for i, e := range errors {
		if _, err := New(i); err == nil || err.Error() != e {
			t.Errorf("Case: %s; Actual: %s; Expected: %#v", i, err, e)
		}
	}
	list := map[string]string{
		"INF,UN":    "0",
		"NEGINF,UN": "0",
		"UNKN,UN":   "1",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestPartialApplication(t *testing.T) {
	exp, err := New("a,b,c,d,+,+,+")
	if err != nil {
		t.Fatal(err)
	}

	bindings := make(map[string]interface{})

	bindings["b"] = 2
	if exp, err = exp.Partial(bindings); err != nil {
		t.Fatalf("Actual: %s; Expected: %#v", err, nil)
	}
	expected := "a,2,c,d,+,+,+"
	if exp.String() != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", exp.String(), expected)
	}

	bindings["d"] = 4
	if exp, err = exp.Partial(bindings); err != nil {
		t.Fatalf("Actual: %s; Expected: %#v", err, nil)
	}
	expected = "a,2,c,4,+,+,+"
	if exp.String() != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", exp.String(), expected)
	}

	bindings["c"] = 3
	if exp, err = exp.Partial(bindings); err != nil {
		t.Fatalf("Actual: %s; Expected: %#v", err, nil)
	}
	expected = "a,9,+"
	if exp.String() != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", exp.String(), expected)
	}

	bindings["a"] = 1
	if exp, err = exp.Partial(bindings); err != nil {
		t.Fatalf("Actual: %s; Expected: %#v", err, nil)
	}
	expected = "10"
	if exp.String() != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", exp.String(), expected)
	}

	value, err := exp.Evaluate(bindings)
	if err != nil {
		t.Errorf("Actual: %#v; Expected: %#v", err, nil)
	}
	if value != 10 {
		t.Errorf("Actual: %#v; Expected: %#v", value, 10)
	}
}

func TestEvaluateWithBindings(t *testing.T) {
	exp, err := New("a,b,c,d,+,+,+")
	if err != nil {
		t.Fatal(err)
	}

	bindings := map[string]interface{}{
		"a": float64(1),
		"b": float64(2),
		"c": float64(3),
		"d": float64(4),
	}
	value, err := exp.Evaluate(bindings)
	if err != nil {
		t.Errorf("Actual: %#v; Expected: %#v", err, nil)
	}
	if value != 10 {
		t.Errorf("Actual: %#v; Expected: %#v", value, 10)
	}
}

func TestEvaluateWithoutBindings(t *testing.T) {
	exp, err := New("a,b,c,d,+,+,+")
	if err != nil {
		t.Fatal(err)
	}

	bindings := make(map[string]interface{})

	value, err := exp.Evaluate(bindings)
	if _, ok := err.(ErrOpenBindings); err == nil || !ok {
		want := []string{"a", "b", "c", "d"}
		t.Errorf("Actual: %#v; Expected: %#v", err, ErrOpenBindings(want))
	}
	if want := float64(0); value != want {
		t.Errorf("Actual: %#v; Expected: %#v", value, want)
	}
}

func TestPartialIgnoresNOWInBindings(t *testing.T) {
	list := map[string]string{
		"1,NOW": "1,NOW",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		exp, err = exp.Partial(map[string]interface{}{"NOW": 12})
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}

func TestEvaluateTREND(t *testing.T) {
	exp, err := New("sam,10,TREND", SecondsPerInterval(1))
	if err != nil {
		t.Fatal(err)
	}

	bindings := map[string]interface{}{
		"sam": []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, math.NaN()},
	}
	value, err := exp.Evaluate(bindings)
	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
	if !math.IsNaN(value) {
		t.Errorf("Actual: %#v; Expected: %#v", value, math.NaN())
	}
}

func TestEvaluateTRENDNotEnoughValues(t *testing.T) {
	exp, err := New("sam,10,TREND", SecondsPerInterval(1))
	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
	bindings := map[string]interface{}{
		"sam": []interface{}{1, 2},
	}
	_, err = exp.Evaluate(bindings)
	if err == nil || err.Error() != "syntax error : TREND operand specifies 10 values, but only 2 available" {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
}

func TestEvaluateTRENDNotBoundToFloatSlice(t *testing.T) {
	exp, err := New("sam,10,TREND", SecondsPerInterval(1))
	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
	bindings := map[string]interface{}{
		"sam": 134,
	}
	_, err = exp.Evaluate(bindings)
	if err == nil || err.Error() != "syntax error : TREND operator requires label but found float64: 134" {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
}

func TestEvaluateTRENDNAN(t *testing.T) {
	exp, err := New("sam,10,TRENDNAN", SecondsPerInterval(1))
	if err != nil {
		t.Fatal(err)
	}

	bindings := map[string]interface{}{
		"sam": []float64{1, 2, math.NaN(), 4, 5, math.NaN(), 7, 8, 9, 10},
	}
	value, err := exp.Evaluate(bindings)
	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
	if value != 5.75 {
		t.Errorf("Actual: %#v; Expected: %#v", value, 5.75)
	}
}

func TestEvaluateTRENDNANNotEnoughValues(t *testing.T) {
	exp, err := New("sam,10,TRENDNAN", SecondsPerInterval(1))
	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
	bindings := map[string]interface{}{
		"sam": []interface{}{1, 2},
	}
	_, err = exp.Evaluate(bindings)
	if err == nil || err.Error() != "syntax error : TRENDNAN operand specifies 10 values, but only 2 available" {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
}

func TestEvaluateTRENDNANNotBoundToFloatSlice(t *testing.T) {
	exp, err := New("sam,10,TRENDNAN", SecondsPerInterval(1))
	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
	bindings := map[string]interface{}{
		"sam": 134,
	}
	_, err = exp.Evaluate(bindings)
	if err == nil || err.Error() != "syntax error : TRENDNAN operator requires label but found float64: 134" {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
}

// evaluate is able to coerce slices of any number type to slices of float64 values

func TestEvaluateTRENDNANSliceOfEmptyInterface(t *testing.T) {
	exp, err := New("sam,10,TRENDNAN", SecondsPerInterval(1))
	if err != nil {
		t.Fatal(err)
	}

	bindings := map[string]interface{}{
		"sam": []interface{}{1, 2, math.NaN(), 4, 5, math.NaN(), 7, 8, 9, 10},
	}
	value, err := exp.Evaluate(bindings)
	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
	if value != 5.75 {
		t.Errorf("Actual: %#v; Expected: %#v", value, 5.75)
	}
}

func TestEvaluateTRENDNANSliceOfFloat64(t *testing.T) {
	exp, err := New("sam,10,TRENDNAN", SecondsPerInterval(1))
	if err != nil {
		t.Fatal(err)
	}

	bindings := map[string]interface{}{
		"sam": []float64{1, 2, math.NaN(), 4, 5, math.NaN(), 7, 8, 9, 10},
	}
	value, err := exp.Evaluate(bindings)
	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
	if value != 5.75 {
		t.Errorf("Actual: %#v; Expected: %#v", value, 5.75)
	}
}

func TestEvaluateTRENDNANSliceOfFloat32(t *testing.T) {
	exp, err := New("sam,10,TRENDNAN", SecondsPerInterval(1))
	if err != nil {
		t.Fatal(err)
	}

	bindings := map[string]interface{}{
		"sam": []float32{1, 2, float32(math.NaN()), 4, 5, float32(math.NaN()), 7, 8, 9, 10},
	}
	value, err := exp.Evaluate(bindings)
	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
	if value != 5.75 {
		t.Errorf("Actual: %#v; Expected: %#v", value, 5.75)
	}
}

func TestEvaluateTRENDNANSliceOfInt(t *testing.T) {
	exp, err := New("sam,10,TRENDNAN", SecondsPerInterval(1))
	if err != nil {
		t.Fatal(err)
	}

	bindings := map[string]interface{}{
		"sam": []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}
	value, err := exp.Evaluate(bindings)
	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
	if value != 5.5 {
		t.Errorf("Actual: %#v; Expected: %#v", value, 5.5)
	}
}

func TestEvaluateTRENDNANSliceOfInt64(t *testing.T) {
	exp, err := New("sam,10,TRENDNAN", SecondsPerInterval(1))
	if err != nil {
		t.Fatal(err)
	}

	bindings := map[string]interface{}{
		"sam": []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}
	value, err := exp.Evaluate(bindings)
	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
	if value != 5.5 {
		t.Errorf("Actual: %#v; Expected: %#v", value, 5.5)
	}
}

func TestEvaluateTRENDNANSliceOfInt32(t *testing.T) {
	exp, err := New("sam,10,TRENDNAN", SecondsPerInterval(1))
	if err != nil {
		t.Fatal(err)
	}

	bindings := map[string]interface{}{
		"sam": []int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}
	value, err := exp.Evaluate(bindings)
	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
	if value != 5.5 {
		t.Errorf("Actual: %#v; Expected: %#v", value, 5.5)
	}
}

// STEPWIDTH

func TestEvaluateSTEPWIDTHDefault(t *testing.T) {
	exp, err := New("STEPWIDTH")
	if err != nil {
		t.Fatal(err)
	}
	value, err := exp.Evaluate(nil)
	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
	if value != 300 {
		t.Errorf("Actual: %#v; Expected: %#v", value, 300)
	}
}

func TestEvaluateSTEPWIDTHCustom(t *testing.T) {
	exp, err := New("STEPWIDTH", SecondsPerInterval(3600))
	if err != nil {
		t.Fatal(err)
	}
	value, err := exp.Evaluate(nil)
	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
	if value != 3600 {
		t.Errorf("Actual: %#v; Expected: %#v", value, 3600)
	}
}

// TIME

func TestEvaluateTIMEWithoutTime(t *testing.T) {
	exp, err := New("TIME")
	if err != nil {
		t.Fatal(err)
	}
	_, err = exp.Evaluate(nil)
	if err == nil || err.Error() != "open bindings: TIME" {
		t.Errorf("Actual: %s; Expected: %#v", err, "open bindings: TIME")
	}
}

func TestEvaluateTIMEWithTime(t *testing.T) {
	exp, err := New("TIME")
	if err != nil {
		t.Fatal(err)
	}
	epoch := 1234567890
	value, err := exp.Evaluate(map[string]interface{}{
		"TIME": epoch,
	})
	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}
	if int(value) != epoch {
		t.Errorf("Actual: %#v; Expected: %#v", int(value), epoch)
	}
}

// LTIME

func TestEvaluateLTIMEWithoutTime(t *testing.T) {
	exp, err := New("LTIME")
	if err != nil {
		t.Fatal(err)
	}
	_, err = exp.Evaluate(nil)
	if err == nil || err.Error() != "open bindings: TIME" {
		t.Errorf("Actual: %s; Expected: %#v", err, "open bindings: TIME")
	}
}

func TestEvaluateLTIMEWithTime(t *testing.T) {
	exp, err := New("LTIME")
	if err != nil {
		t.Fatal(err)
	}

	epoch := 1234567890
	utcTime := time.Unix(int64(epoch), 0)
	_, offset := utcTime.Zone()
	expected := epoch + offset

	value, err := exp.Evaluate(map[string]interface{}{
		"TIME": epoch,
	})

	if err != nil {
		t.Errorf("Actual: %s; Expected: %#v", err, nil)
	}

	if int(value) != expected {
		t.Errorf("Actual: %#v; Expected: %#v", int(value), expected)
	}
}

// MEDIAN

func TestNewExpressionMEDIAN(t *testing.T) {
	errors := map[string]string{
		"1,2,3,-1,MEDIAN":     "syntax error : MEDIAN operator requires positive finite integer: -1",
		"1,2,3,0,MEDIAN":      "syntax error : MEDIAN operator requires positive finite integer: 0",
		"1,2,3,4,MEDIAN":      "syntax error : MEDIAN 4 items, but only 3 on stack",
		"1,2,3,INF,MEDIAN":    "syntax error : MEDIAN operator requires positive finite integer: +Inf",
		"1,2,3,NEGINF,MEDIAN": "syntax error : MEDIAN operator requires positive finite integer: -Inf",
	}
	for i, e := range errors {
		if _, err := New(i); err == nil || err.Error() != e {
			t.Errorf("Case: %s; Actual: %s; Expected: %#v", i, err, e)
		}
	}
	list := map[string]string{
		// "a,b,c,3,MEDIAN": "a,b,c,3,MEDIAN", // cannot sort variables

		// one item
		"13,1,MEDIAN": "13",
		"a,1,MEDIAN":  "a", // pin-hole optimization

		// two items -- average
		"a,b,c,d,e,f,13,42,2,MEDIAN": "a,b,c,d,e,f,27.5",
		"42,13,2,MEDIAN":             "27.5",

		// three items -- middle
		"42,666,13,3,MEDIAN": "42",
		// four items -- average of middle
		"1,1,2,3,4,MEDIAN": "1.5",
		// five items -- middle
		"3,2,5,1,4,5,MEDIAN": "3",
		//
		"13,a,ISINF,2,MEDIAN": "13,a,ISINF,2,MEDIAN",
		"67,42,13,2,MEDIAN,-": "39.5",
	}
	for input, output := range list {
		exp, err := New(input)
		if err != nil {
			t.Fatalf("Case: %s; Actual: %#v; Expected: %#v", input, err, nil)
		}
		if exp.String() != output {
			t.Errorf("Case: %s; Actual: %#v; Expected: %#v", input, exp.String(), output)
		}
	}
}
