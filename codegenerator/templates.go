package main

import "text/template"

var (
	resultStruct = "type Result struct {\n" +
		"\tError    string      `json:\"error\"`\n" +
		"\tResponse interface{} `json:\"response\"`\n}"

	serveHTTP = template.Must(template.New("serveHTTP").Parse(`
func (h *{{.StructName}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	{{range .Methods}}
	case "{{.URL}}":
		h.Handler{{.MethodName}}(w, r)
	{{end}}
	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("{\"error\": \"unknown method\"}"))
	}
}
`))

	doJobTemplate = template.Must(template.New("doJobTemplate").Parse(`
func (h *{{.StructName}}) Handler{{.MethodName}}(w http.ResponseWriter, r *http.Request) {
	//проверка метода
	{{if .Post}}
	if r.Method != "{{.Method}}" {
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
	{{else}}
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
	{{end}}
	{{if .Auth}}
	if token := r.Header.Get("X-Auth"); token == "" {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("{\"error\": \"unauthorized\"}"))
		return
	}
	{{end}}
	// заполнение структуры params
	// валидирование параметров
	params := {{.MethodParamsStructName}}{}
	{{range .ParamsStructNameFields}}
	{{.Code}}
	{{end}}
	response, err := h.{{.MethodName}}(context.Background(), params)
	if err != nil {
		//log.Printf("Error in h.{{.MethodName}}\n")
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
`))

	paramsCode = template.Must(template.New("paramsCode").Parse(`{
		{{ if .ChangeName }}URLname := "{{.NewName}}"
		{{else}}URLname := strings.ToLower("{{.ParamName}}")
		{{end}}
		_, ok := q[URLname]
		var value string
		if !ok {
			{{if .Required}}
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must me not empty\"}", URLname)))
			return
			{{end}}
			{{if .Default}}
			value = "{{.DefaultValue}}"
			{{end}}
		} else {
			value = q[URLname]
		}
		{{if .Enum}}checkEnum := false
		{{range .EnumValues}}
		checkEnum = checkEnum || value == "{{.Value}}"
		{{end}}
		if !checkEnum {
			w.WriteHeader(http.StatusBadRequest)
			var str []string
			{{range .EnumValues}}
			str = append(str, "{{.Value}}")
			{{end}}
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must be one of [%s]\"}", URLname, strings.Join(str, ", "))))
			return
		}{{end}}
		{{if .MinCondition}}
		{{if .IsTypeInt}}
		if val, err := strconv.Atoi(value); err!=nil || val < {{.Min}}{
			if err != nil {
				//log.Printf("Can not cast value %s to int: %s\n", value, err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must be int\"}", URLname)))
			} else {
				//log.Printf("wrong val: %d < {{.Min}} \n", val)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must be >= {{.Min}}\"}", URLname)))
			}
			return
		}
		{{else}}
		if val := len(value); val < {{.Min}}{
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s len must be >= {{.Min}}\"}", URLname)))
			return
		}
		{{end}}
		{{end}}
		{{if .MaxCondition}}
		{{if .IsTypeInt}}
		if val, err := strconv.Atoi(value); err!=nil || val > {{.Max}}{
			if err != nil {
				//log.Printf("Can not cast value %s to int: %s\n", value, err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must be int\"}", URLname)))
			} else {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("{\"error\": \"%s must be <= {{.Max}}\"}", URLname)))
			}
			return
		}
		{{else}}
		if val := len(value); val > {{.Max}}{
			//log.Printf("wrong len: %s\n", value)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("{\"error\": \"%s len must be <= {{.Max}}\"}", URLname)))
			return
		}
		{{end}}
		{{end}}
		{{if .IsTypeInt}}
		var err error
		params.{{.ParamName}}, err = strconv.Atoi(value)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			//log.Printf("can not casting %s to int\n", value)
			return
		}
		{{else}}
		params.{{.ParamName}} = value
		{{end}}
	}
`))

	imp = `import (
	"context"
	"io/ioutil"
	"fmt"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"strconv"
)`
)
