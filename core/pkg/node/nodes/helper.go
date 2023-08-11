package nodes

import (
	"encoding/json"
	"strings"

	"github.com/xpsl/govaluate"
)

func Evaluate(expr string, dataSlice ...map[string]interface{}) (interface{}, error) {
	expr = trim(expr, "$formData")
	expr = trim(expr, "$variable")

	expression, err := govaluate.NewEvaluableExpression(expr)
	if err != nil {
		return nil, err
	}

	parameters := make(map[string]interface{})
	for _, data := range dataSlice {
		for name, value := range data {
			if strings.Contains(expr, name) {
				parameters[name] = value
			}
		}
	}

	return expression.Evaluate(parameters)
}

func trim(str, sub string) string {
	if i := strings.Index(str, sub); i > 0 {
		for j := i; j < len(str); j++ {
			if str[j] == '.' {
				str = strings.ReplaceAll(str, str[i:j+1], "")
			}
		}
	}

	return str
}

func GuessCommunal(src map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(src))
	for name, value := range src {
		var any interface{}
		err := json.Unmarshal([]byte(value), &any)
		if err == nil {
			result[name] = any
		} else {
			result[name] = value
		}
	}

	return result
}
