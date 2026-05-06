package util

import (
	"fmt"
	"sort"
	"strconv"

	"gopkg.in/yaml.v3"
)

// ParseClassNames extracts YOLO class names from data.yaml text.
func ParseClassNames(dataYaml []byte) []string {
	var root map[string]interface{}
	if err := yaml.Unmarshal(dataYaml, &root); err != nil {
		return nil
	}
	names, ok := root["names"]
	if !ok {
		return nil
	}
	switch v := names.(type) {
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, x := range v {
			out = append(out, fmt.Sprint(x))
		}
		return out
	case map[string]interface{}:
		keys := make([]int, 0, len(v))
		for k := range v {
			i, err := strconv.Atoi(k)
			if err == nil {
				keys = append(keys, i)
			}
		}
		sort.Ints(keys)
		out := make([]string, len(keys))
		for i, k := range keys {
			out[i] = fmt.Sprint(v[strconv.Itoa(k)])
		}
		return out
	default:
		return nil
	}
}
