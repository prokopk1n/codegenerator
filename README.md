# Кодогенерация Go
Анализ go программы и генерация по ней соответствующего API.

#### Установка и запуск
    git clone https://github.com/prokopk1n/codegenerator
    cd codegenerator
    go build -o codegenerator.exe .
    ./codegeneratot.exe {source-go-file} {dest-go-file}
#### Тестирование кодогенератора
    ./codegenerator.exe ../tests/api.go ../tests/api_handlers.go
    cd ../tests
    go test -v

#### Подробности реализации
Для методов-обработчиков доступны следующие параметры, следующие за тегом apigen:api:
- url - оработчиком какого url является данный метод
- auth - требуется ли аутентификация
- method - http.GET или http.POST
Для структур-параметров доступны следующие метки тега apivalidator:
- required - поле не должно быть пустым (не должно иметь значение по-умолчанию)
- paramname - если указано - то брать из параметра с этим именем, иначе lowercase от имени
- enum - "одно из"
- default - если указано и приходит пустое значение (значение по-умолчанию) - устанавливать то что написано указано в default
- min - >= X для типа int, для строк len(str) >=
- max - <= X для типа int

#### Пример генерации
Для каждого метода вида:
```
    // apigen:api {"url": "/user/profile", "auth": false}
    func (srv *MyApi) Profile(ctx context.Context, in ProfileParams) (*User, error) {
        ...
    }
```
В соответствии с apigen:api, а также структурой-параметром:
```
type ProfileParams struct {
	Login string `apivalidator:"required"`
}
```

Генерируется соответствующая обертка, предназначенная для обработки http-запросов.<br>
Как можно увидеть ниже, в соответствии с тегом apivalidator у поля структуры в коде осуществляется проверка наличия параметра "login" в запросе:
```
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
```
В конечном итоге для всех сгенерированных обработчиков генерируется метод ServeHTTP вида:
```
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
```
