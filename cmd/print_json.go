package cmd

import (
	"encoding/json"
	"fmt"
)

func printJSON(obj interface{}) {
	json, _ := json.MarshalIndent(obj, "", "  ")
	fmt.Println(string(json))
}
