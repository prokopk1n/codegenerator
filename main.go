package main

// это программа для которой ваш кодогенератор будет писать код
// запускать через go test -v, как обычно

// этот код закомментирован чтобы он не светился в тестовом покрытии

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func main() {
	// будет вызван метод ServeHTTP у структуры MyApi
	http.Handle("/user/", NewMyApi())

	fmt.Println("starting server at :8080")
	go http.ListenAndServe(":8080", nil)
	time.Sleep(5 * time.Second)
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("POST", "http://localhost:8080/user/create?login=new_moderator&age=ten&status=moderator&full_name=Ivan_Ivanov", nil)
	req.Header.Add("X-Auth", "100500")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("request error: %v", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	result := make(map[string]interface{})
	json.Unmarshal(body, &result)

	for key, value := range result {
		fmt.Printf("key %+v value %+v\n", key, value)
	}
	fmt.Println("code", resp.StatusCode)

	time.Sleep(10000 * time.Second)
}
