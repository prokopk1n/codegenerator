package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"
)

type Params struct {
	//нужно ли заменять имя в URL
	ChangeName bool
	NewName    string
	//имя в структуре
	ParamName    string
	Required     bool
	Default      bool
	DefaultValue string
	Enum         bool
	EnumValues   []struct {
		Value string
	}
	MinCondition bool
	Min          int
	MaxCondition bool
	Max          int
	//если int, то тут true
	Type bool
	Post bool
}

type DoJobParams struct {
	StructName string
	MethodName string
	//POST, GET
	Method           string
	Auth             bool
	StructParamsName string
	Params           []struct {
		Code string
	}
	Post bool
}

type ServeHTTPParams struct {
	StructName string
	Methods    []struct {
		URL        string
		MethodName string
	}
}

var (
	resultStruct = "type Result struct {\n" +
		"\tError    string      `json:\"error\"`\n" +
		"\tResponse interface{} `json:\"response\"`\n}"

	serveHTTP = template.Must(template.New("serveHTTP").Parse(`
func (h *{{.StructName}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	{{range .Methods}}
	case "{{.URL}}":
		h.handler{{.MethodName}}(w, r)
	{{end}}
	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("{\"error\": \"unknown method\"}"))
	}
}
`))

	doJob = template.Must(template.New("doJob").Parse(`
func (h *{{.StructName}}) handler{{.MethodName}}(w http.ResponseWriter, r *http.Request) {
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
	params := {{.StructParamsName}}{}
	{{range .Params}}
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
		{{if .Type}}
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
		{{if .Type}}
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
		{{if .Type}}
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

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(os.Args[2])

	fmt.Fprintln(out, `package `+node.Name.Name)
	fmt.Fprintln(out) // empty line

	fmt.Fprintln(out, imp)
	fmt.Fprintln(out) // empty line

	fmt.Fprintln(out, resultStruct)

	serveHTTPmap := make(map[string]ServeHTTPParams)

	for _, f := range node.Decls {
		post := false
		g, ok := f.(*ast.FuncDecl)
		if !ok {
			fmt.Printf("SKIP %T is not *ast.FunDecl\n", f)
			continue
		}

		if g.Doc == nil {
			fmt.Printf("SKIP %T has not comment\n", g)
			continue
		}

		serveHTTPParams := serveHTTPmap[g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name]

		doJobParams := DoJobParams{}
		needCodegen := false
		for _, comment := range g.Doc.List {
			if !strings.HasPrefix(comment.Text, "// apigen:api") {
				continue
			}
			needCodegen = true
			value := make(map[string]interface{})
			if err := json.Unmarshal([]byte("{"+strings.Split(comment.Text, "{")[1]), &value); err != nil {
				fmt.Printf("Can not parse comment %s \n%s\n", "{"+strings.Split(comment.Text, "{")[1], err)
			}
			if result, ok := value["url"]; !ok {
				fmt.Printf("Can not find url field\n")
				return
			} else {
				serveHTTPParams.Methods = append(serveHTTPParams.Methods,
					struct {
						URL        string
						MethodName string
					}{result.(string), g.Name.Name})
			}

			if result, ok := value["auth"]; !ok {
				fmt.Printf("Can not find auth field\n")
				return
			} else {
				doJobParams.Auth = result.(bool)
			}

			if _, ok := value["method"]; !ok {
				doJobParams.Method = http.MethodGet
			} else {
				doJobParams.Method = http.MethodPost
				doJobParams.Post = true
				post = true
			}

		}
		if !needCodegen {
			fmt.Printf("Function %T does not need codegen\n", f)
			continue
		}
		serveHTTPParams.StructName = g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		doJobParams.MethodName = g.Name.Name
		doJobParams.StructName = g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		doJobParams.StructParamsName = g.Type.Params.List[1].Type.(*ast.Ident).Name

		structParamsName := g.Type.Params.List[1].Type.(*ast.Ident).Name

		buf := new(bytes.Buffer)
		var currStruct *ast.StructType

		serveHTTPmap[g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name] = serveHTTPParams

		//ищем структуру, которая является параметром функции
	FIND_STRUCT:
		for _, f_new := range node.Decls {
			g, ok := f_new.(*ast.GenDecl)
			if !ok {
				fmt.Printf("SKIP %T is not *ast.GenDecl\n", f)
				continue
			}
			for _, spec := range g.Specs {
				currType, ok := spec.(*ast.TypeSpec)
				if !ok {
					fmt.Printf("SKIP %T is not ast.TypeSpec\n", spec)
					continue
				}

				if currType.Name.Name != structParamsName {
					continue
				}

				currStruct, ok = currType.Type.(*ast.StructType)
				if !ok {
					fmt.Printf("SKIP %T is not ast.StructType\n", currStruct)
					continue
				}

				break FIND_STRUCT
			}
		}

		//проходимся по полям структуры
		for _, field := range currStruct.Fields.List {
			params := Params{}
			params.Post = post
			buf.Reset()
			if field.Type.(*ast.Ident).Name == "int" { // types.ExprString(field.Type)
				params.Type = true
			}
			params.ParamName = field.Names[0].Name
			if field.Tag == nil {
				if err := paramsCode.Execute(buf, params); err != nil {
					fmt.Printf("Error in paramsCode.Execute: %s\n", err)
				}
				doJobParams.Params = append(doJobParams.Params, struct{ Code string }{buf.String() + "\n"})
			} else {
				tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
				apiTag := tag.Get("apivalidator")
				for _, substr := range strings.Split(apiTag, ",") {
					if strings.Contains(substr, "required") {
						params.Required = true
						continue
					}
					if strings.Contains(substr, "paramname") {
						params.ChangeName = true
						params.NewName = strings.Split(substr, "=")[1]
						continue
					}
					if strings.Contains(substr, "enum") {
						params.Enum = true
						for _, value := range strings.Split(strings.Split(substr, "=")[1], "|") {
							params.EnumValues = append(params.EnumValues, struct{ Value string }{value})
						}
						continue
					}
					if strings.Contains(substr, "default") {
						params.Default = true
						params.DefaultValue = strings.Split(substr, "=")[1]
						continue
					}
					if strings.Contains(substr, "min") {
						params.MinCondition = true
						params.Min, err = strconv.Atoi(strings.Split(substr, "=")[1])
						if err != nil {
							fmt.Printf("can not parse min value to int %s\n", strings.Split(substr, "=")[1])
							params.MinCondition = false
						}
						continue
					}
					if strings.Contains(substr, "max") {
						params.MaxCondition = true
						params.Max, err = strconv.Atoi(strings.Split(substr, "=")[1])
						if err != nil {
							fmt.Printf("can not parse max value to int %s\n", strings.Split(substr, "=")[1])
							params.MaxCondition = false
						}
						continue
					}
				}
				if err := paramsCode.Execute(buf, params); err != nil {
					fmt.Printf("Error in paramsCode.Execute: %s\n", err)
				}
				doJobParams.Params = append(doJobParams.Params, struct{ Code string }{buf.String() + "\n"})
			}
		}

		buf.Reset()
		if err := doJob.Execute(out, doJobParams); err != nil {
			fmt.Printf("Error in doJob.Execut: %s\n", err)
		}
		fmt.Fprintln(out) // empty line

	}

	for _, value := range serveHTTPmap {
		if err := serveHTTP.Execute(out, value); err != nil {
			log.Printf("Error in serveHTTP.Execute: %s\n", err)
		}
		fmt.Fprintln(out) // empty line
	}
}
