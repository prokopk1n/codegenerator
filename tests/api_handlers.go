package tests

import (
	"context"
	"io/ioutil"
	"fmt"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"strconv"
)

type Result struct {
	Error    string      `json:"error"`
	Response interface{} `json:"response"`
}

func (h *MyApi) HandlerProfile(w http.ResponseWriter, r *http.Request) {
	//проверка метода
	
	q := make(map[string]string)
	if r.Method == "GET" {
		query := r.URL.Query()
		for key, value := range query {
			q[key] = value[0]
		}
	} else {
		b, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		query := string(b)
		for _, str := range strings.Split(query, "&") {
			if str == "" {
				continue
			} else if len(strings.Split(str, "=")) == 1 {
				w.WriteHeader(http.StatusBadRequest)
			}
			q[strings.Split(str, "=")[0]] = strings.Split(str, "=")[1]
		}
	}
	
	
	// заполнение структуры params
	// валидирование параметров
	params := ProfileParams{}
	
	{
		URLname := strings.ToLower("Login")
		
		_, ok := q[URLname]
		var value string
		if !ok {
			
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must me not empty\"}", URLname)))
			return
			
			
		} else {
			value = q[URLname]
		}
		
		
		
		
		params.Login = value
		
	}


	
	response, err := h.Profile(context.Background(), params)
	if err != nil {
		//log.Printf("Error in h.Profile\n")
		if errBody, ok := err.(ApiError); !ok {
			w.WriteHeader(http.StatusInternalServerError)
			//log.Printf("{\"error\": \"%s\"}", err.Error())
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err.Error())))
			return
		} else {
			w.WriteHeader(errBody.HTTPStatus)
			//log.Printf("{\"error\": \"%s\"}", errBody.Error())
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", errBody.Error())))
		return
		}
	}
	jsonResponse, err := json.Marshal(Result{"", response})
	if err != nil {
		//log.Printf("Error in Marshal %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		log.Printf("Ok\n")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

func (h *MyApi) HandlerCreate(w http.ResponseWriter, r *http.Request) {
	//проверка метода
	
	if r.Method != "POST" {
		//log.Printf("Wrong Method\n")
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("{\"error\": \"bad method\"}"))
		return
	}
	q := make(map[string]string)
	b, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	query := string(b)
	for _, str := range strings.Split(query, "&") {
		if str == "" {
			continue
		} else if len(strings.Split(str, "=")) == 1 {
			w.WriteHeader(http.StatusBadRequest)
		}
		q[strings.Split(str, "=")[0]] = strings.Split(str, "=")[1]
	}
	
	
	if token := r.Header.Get("X-Auth"); token == "" {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("{\"error\": \"unauthorized\"}"))
		return
	}
	
	// заполнение структуры params
	// валидирование параметров
	params := CreateParams{}
	
	{
		URLname := strings.ToLower("Login")
		
		_, ok := q[URLname]
		var value string
		if !ok {
			
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must me not empty\"}", URLname)))
			return
			
			
		} else {
			value = q[URLname]
		}
		
		
		
		if val := len(value); val < 10{
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s len must be >= 10\"}", URLname)))
			return
		}
		
		
		
		
		params.Login = value
		
	}


	
	{
		URLname := "full_name"
		
		_, ok := q[URLname]
		var value string
		if !ok {
			
			
		} else {
			value = q[URLname]
		}
		
		
		
		
		params.Name = value
		
	}


	
	{
		URLname := strings.ToLower("Status")
		
		_, ok := q[URLname]
		var value string
		if !ok {
			
			
			value = "user"
			
		} else {
			value = q[URLname]
		}
		checkEnum := false
		
		checkEnum = checkEnum || value == "user"
		
		checkEnum = checkEnum || value == "moderator"
		
		checkEnum = checkEnum || value == "admin"
		
		if !checkEnum {
			w.WriteHeader(http.StatusBadRequest)
			var str []string
			
			str = append(str, "user")
			
			str = append(str, "moderator")
			
			str = append(str, "admin")
			
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must be one of [%s]\"}", URLname, strings.Join(str, ", "))))
			return
		}
		
		
		
		params.Status = value
		
	}


	
	{
		URLname := strings.ToLower("Age")
		
		_, ok := q[URLname]
		var value string
		if !ok {
			
			
		} else {
			value = q[URLname]
		}
		
		
		
		if val, err := strconv.Atoi(value); err!=nil || val < 0{
			if err != nil {
				//log.Printf("Can not cast value %s to int: %s\n", value, err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must be int\"}", URLname)))
			} else {
				//log.Printf("wrong val: %d < 0 \n", val)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must be >= 0\"}", URLname)))
			}
			return
		}
		
		
		
		
		if val, err := strconv.Atoi(value); err!=nil || val > 128{
			if err != nil {
				//log.Printf("Can not cast value %s to int: %s\n", value, err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must be int\"}", URLname)))
			} else {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must be <= 128\"}", URLname)))
			}
			return
		}
		
		
		
		var err error
		params.Age, err = strconv.Atoi(value)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			//log.Printf("can not casting %s to int\n", value)
			return
		}
		
	}


	
	response, err := h.Create(context.Background(), params)
	if err != nil {
		//log.Printf("Error in h.Create\n")
		if errBody, ok := err.(ApiError); !ok {
			w.WriteHeader(http.StatusInternalServerError)
			//log.Printf("{\"error\": \"%s\"}", err.Error())
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err.Error())))
			return
		} else {
			w.WriteHeader(errBody.HTTPStatus)
			//log.Printf("{\"error\": \"%s\"}", errBody.Error())
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", errBody.Error())))
		return
		}
	}
	jsonResponse, err := json.Marshal(Result{"", response})
	if err != nil {
		//log.Printf("Error in Marshal %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		log.Printf("Ok\n")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

func (h *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	
	case "/user/profile":
		h.HandlerProfile(w, r)
	
	case "/user/create":
		h.HandlerCreate(w, r)
	
	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("{\"error\": \"unknown method\"}"))
	}
}


func (h *OtherApi) HandlerCreate(w http.ResponseWriter, r *http.Request) {
	//проверка метода
	
	if r.Method != "POST" {
		//log.Printf("Wrong Method\n")
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("{\"error\": \"bad method\"}"))
		return
	}
	q := make(map[string]string)
	b, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	query := string(b)
	for _, str := range strings.Split(query, "&") {
		if str == "" {
			continue
		} else if len(strings.Split(str, "=")) == 1 {
			w.WriteHeader(http.StatusBadRequest)
		}
		q[strings.Split(str, "=")[0]] = strings.Split(str, "=")[1]
	}
	
	
	if token := r.Header.Get("X-Auth"); token == "" {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("{\"error\": \"unauthorized\"}"))
		return
	}
	
	// заполнение структуры params
	// валидирование параметров
	params := OtherCreateParams{}
	
	{
		URLname := strings.ToLower("Username")
		
		_, ok := q[URLname]
		var value string
		if !ok {
			
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must me not empty\"}", URLname)))
			return
			
			
		} else {
			value = q[URLname]
		}
		
		
		
		if val := len(value); val < 3{
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s len must be >= 3\"}", URLname)))
			return
		}
		
		
		
		
		params.Username = value
		
	}


	
	{
		URLname := "account_name"
		
		_, ok := q[URLname]
		var value string
		if !ok {
			
			
		} else {
			value = q[URLname]
		}
		
		
		
		
		params.Name = value
		
	}


	
	{
		URLname := strings.ToLower("Class")
		
		_, ok := q[URLname]
		var value string
		if !ok {
			
			
			value = "warrior"
			
		} else {
			value = q[URLname]
		}
		checkEnum := false
		
		checkEnum = checkEnum || value == "warrior"
		
		checkEnum = checkEnum || value == "sorcerer"
		
		checkEnum = checkEnum || value == "rouge"
		
		if !checkEnum {
			w.WriteHeader(http.StatusBadRequest)
			var str []string
			
			str = append(str, "warrior")
			
			str = append(str, "sorcerer")
			
			str = append(str, "rouge")
			
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must be one of [%s]\"}", URLname, strings.Join(str, ", "))))
			return
		}
		
		
		
		params.Class = value
		
	}


	
	{
		URLname := strings.ToLower("Level")
		
		_, ok := q[URLname]
		var value string
		if !ok {
			
			
		} else {
			value = q[URLname]
		}
		
		
		
		if val, err := strconv.Atoi(value); err!=nil || val < 1{
			if err != nil {
				//log.Printf("Can not cast value %s to int: %s\n", value, err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must be int\"}", URLname)))
			} else {
				//log.Printf("wrong val: %d < 1 \n", val)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must be >= 1\"}", URLname)))
			}
			return
		}
		
		
		
		
		if val, err := strconv.Atoi(value); err!=nil || val > 50{
			if err != nil {
				//log.Printf("Can not cast value %s to int: %s\n", value, err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must be int\"}", URLname)))
			} else {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must be <= 50\"}", URLname)))
			}
			return
		}
		
		
		
		var err error
		params.Level, err = strconv.Atoi(value)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			//log.Printf("can not casting %s to int\n", value)
			return
		}
		
	}


	
	response, err := h.Create(context.Background(), params)
	if err != nil {
		//log.Printf("Error in h.Create\n")
		if errBody, ok := err.(ApiError); !ok {
			w.WriteHeader(http.StatusInternalServerError)
			//log.Printf("{\"error\": \"%s\"}", err.Error())
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err.Error())))
			return
		} else {
			w.WriteHeader(errBody.HTTPStatus)
			//log.Printf("{\"error\": \"%s\"}", errBody.Error())
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", errBody.Error())))
		return
		}
	}
	jsonResponse, err := json.Marshal(Result{"", response})
	if err != nil {
		//log.Printf("Error in Marshal %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		log.Printf("Ok\n")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

func (h *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	
	case "/user/create":
		h.HandlerCreate(w, r)
	
	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("{\"error\": \"unknown method\"}"))
	}
}

