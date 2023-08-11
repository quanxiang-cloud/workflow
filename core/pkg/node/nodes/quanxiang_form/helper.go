package quanxiangform

import (
	"strings"
)

type FlowFormValueRule struct {
	ValueFrom string      `json:"valueFrom,omitempty"`
	ValueOf   interface{} `json:"valueOf,omitempty"`
}

type Expr struct {
	Key   string
	Value string
	Op    string
}

func ParseExpr(str string) []*Expr {
	var subsection = func(symbol string, strs ...string) []string {
		result := make([]string, 0)
		for _, str := range strs {
			result = append(result, strings.Split(str, symbol)...)
		}

		return result
	}

	var parseExpr = func(str string) *Expr {
		str = strings.ReplaceAll(str, " ", "")

		var sub []string
		var op string
		switch {
		case strings.Contains(str, "=="):
			sub = strings.Split(str, "==")
			op = "=="
		case strings.Contains(str, "!="):
			sub = strings.Split(str, "!=")
			op = "!="
		case strings.Contains(str, ">"):
			sub = strings.Split(str, ">")
			op = ">"
		case strings.Contains(str, ">="):
			sub = strings.Split(str, ">=")
			op = ">="
		case strings.Contains(str, "<"):
			sub = strings.Split(str, "<")
			op = "<"
		case strings.Contains(str, "<="):
			sub = strings.Split(str, "<=")
			op = "<="
		}

		if len(sub) == 2 {
			return &Expr{
				Key:   sub[0],
				Value: sub[1],
				Op:    op,
			}
		}

		return nil
	}

	strs := subsection("||", str)
	strs = subsection("&&", strs...)

	result := make([]*Expr, 0, len(strs))
	for _, str := range strs {
		e := parseExpr(str)
		if e != nil {
			result = append(result, e)
		}
	}

	return result
}
