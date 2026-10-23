// Command elk-facts prints the system facts collected by Elk.
//
// The output matches the list format of subscription-manager facts: one
// sorted "key: value" pair per line, with empty values rendered as Unknown.
package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/m-horky/elk/internal/candlepin"
	"github.com/m-horky/elk/pkg/facts"
)

func main() {
	collected := facts.Collect(context.Background())

	all := candlepin.NewFactsDTO(collected)
	keys := make([]string, 0, len(all))

	for key := range all {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		value := all[key]
		if value == "" {
			value = "Unknown"
		}

		fmt.Printf("%s: %s\n", key, value)
	}
}
