package main

import (
	"encoding/json"
	"fmt"
)

type thing struct {
	Stuff map[string]interface{} `json:"service_map"`
}

type data struct {
	Data map[string]string `json:"data"`
}

func main() {
	data := data{
		Data: map[string]string{
			"service_map": "{\"jupyter-chjncprn-project\": \"http://anon-chjncprn-project.main-namespace:8888\", \"jupyter-xzvd42np-project\": \"http://anon-xzvd42np-project.main-namespace:8888\", \"jupyter-admin-myjupyter\": \"http://admin-myjupyter.main-namespace:8888\", \"jupyter-admin-notjhub\": \"http://admin-notjhub.main-namespace:8888\"}",
		},
	}

	jsonData, _ := json.Marshal(data.Data)

	stuff := thing{}
	json.Unmarshal(jsonData, &stuff)
	fmt.Println(string(jsonData))
	fmt.Println(stuff)

	// stuff2 := make(map[string]string)
	json.Unmarshal([]byte(data.Data["service_map"]), &stuff.Stuff)
	fmt.Println(string(jsonData))
	fmt.Println(stuff)
}
