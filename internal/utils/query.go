package utils

import (
	"fmt"
	"strings"
)

func ParseQuery(query string) map[string]any {
	dict := make(map[string]any)
	for couple := range strings.SplitSeq(query, "&"){
		key, value, f := strings.Cut(couple, "=")
		if f {
			dict[key] = value
		}
	}	

	return dict
}

func EncodeQuery(data map[string]any) (encoded string) {
	if len(data) > 0 {
		couples := []string{"?"}
		for key, value := range data {
			couples = append(couples, fmt.Sprintf("%v=%v", key, value))
		}
		encoded = strings.Join(couples, "&")
	}
	return encoded
}
