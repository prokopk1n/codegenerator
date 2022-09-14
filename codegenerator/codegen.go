package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

//Params описывает параметры метода,
//который будет вызван после валидации параметров из соответствующего handlera
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
	IsTypeInt bool
	Post      bool
	Code      string
}

// DoJobParams структура предназначена для создания нового handlera (имя = handler + MethodName, который будет методом
// уже существующей структуры StructName. Этот handler сначала валидирует параметры, формирует структуру-параметр,
// а затем вызывает соответствующий метод структуры StructName.MethodName
type DoJobParams struct {
	StructName string
	MethodName string
	//POST, GET
	Method string
	Auth   bool
	//имя структуры-параметра метода StructName.MethodName
	MethodParamsStructName string
	//поля структуры-параметра
	ParamsStructNameFields []*Params
	Post                   bool
}

// ServeHTTPParams структура хранит имя структуры,
// для которой нужно создать новые методы, указанные в срезе Methods
// URL - какой адрес обрабатывать, MethodName - каким методом этой структуры (уже написанным)
type ServeHTTPParams struct {
	StructName string
	Methods    []struct {
		URL        string
		MethodName string
		DoJob      *DoJobParams
	}
}

//FindStruct ищет имя заданной структуры в распаршенном файле
func FindStruct(structParamsName string, node *ast.File) (currStruct *ast.StructType) {
FIND_STRUCT:
	for _, f_new := range node.Decls {
		g, ok := f_new.(*ast.GenDecl)
		if !ok {
			fmt.Printf("SKIP %T is not *ast.GenDecl\n", f_new)
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
	return
}

//Заполняем массив Params полями структуры-параметра
func getStructFields(currStruct *ast.StructType, post bool) []*Params {
	var err error
	structFields := make([]*Params, 0)
	for _, field := range currStruct.Fields.List {
		params := Params{}
		params.Post = post
		if field.Type.(*ast.Ident).Name == "int" { // types.ExprString(field.Type)
			params.IsTypeInt = true
		}
		params.ParamName = field.Names[0].Name
		if field.Tag != nil {
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
		}
		structFields = append(structFields, &params)
	}
	return structFields
}

func GoThroughDecls(node *ast.File) (map[string]ServeHTTPParams, error) {
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

		structName := g.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		serveHTTPParams := serveHTTPmap[structName]

		doJobParams := &DoJobParams{}
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
				return nil, fmt.Errorf("Can not find url field")
			} else {
				serveHTTPParams.Methods = append(serveHTTPParams.Methods,
					struct {
						URL        string
						MethodName string
						DoJob      *DoJobParams
					}{result.(string), g.Name.Name, doJobParams})
			}

			if result, ok := value["auth"]; !ok {
				fmt.Printf("Can not find auth field\n")
				return nil, fmt.Errorf("Can not find auth field\n")
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
		serveHTTPParams.StructName = structName
		serveHTTPmap[structName] = serveHTTPParams

		doJobParams.MethodName = g.Name.Name
		doJobParams.StructName = structName
		structParamsName := g.Type.Params.List[1].Type.(*ast.Ident).Name
		doJobParams.MethodParamsStructName = structParamsName

		//ищем структуру, которая является параметром функции
		currStruct := FindStruct(structParamsName, node)
		if currStruct == nil {
			return nil, fmt.Errorf("Can not find struct: %s", structParamsName)
		}

		doJobParams.ParamsStructNameFields = getStructFields(currStruct, post)
		//проходимся по полям структуры
	}
	return serveHTTPmap, nil
}

func GenerateCode(out io.Writer, serveHTTPmap map[string]ServeHTTPParams, packageName string) error {
	fmt.Fprintln(out, "package "+packageName)
	fmt.Fprintln(out) // empty line

	fmt.Fprintln(out, imp)
	fmt.Fprintln(out) // empty line

	fmt.Fprintln(out, resultStruct)

	for _, value := range serveHTTPmap {
		for _, method := range value.Methods {
			doJob := method.DoJob
			for _, params := range doJob.ParamsStructNameFields {
				buf := new(bytes.Buffer)
				err := paramsCode.Execute(buf, params)
				if err != nil {
					return fmt.Errorf("Error inside template paramsCode: %v", err)
				}
				params.Code = buf.String() + "\n"
			}
			if err := doJobTemplate.Execute(out, doJob); err != nil {
				return fmt.Errorf("Error in  doJobTemplate.Execute: %v\n", err)
			}
		}
		if err := serveHTTP.Execute(out, value); err != nil {
			return fmt.Errorf("Error in serveHTTP.Execute: %s\n", err)
		}
		fmt.Fprintln(out) // empty line
	}
	return nil
}
