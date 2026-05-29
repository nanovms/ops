package cmd

import (
	"encoding/json"
	"fmt"
)

func printJSON(obj any) {
	json, _ := json.MarshalIndent(obj, "", "  ")
	fmt.Println(string(json))
}
