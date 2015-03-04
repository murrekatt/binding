package binding

import(
	"mime/multipart"
	"reflect"
)

// Takes values from the form data and puts them into a struct
func mapForm(formStruct reflect.Value, form map[string][]string,
	formfile map[string][]*multipart.FileHeader, errors Errors) {

	if formStruct.Kind() == reflect.Ptr {
		formStruct = formStruct.Elem()
	}

	for i := 0; i < formStruct.Type().NumField(); i++ {
		typeField := formStruct.Type().Field(i)
		structField := formStruct.Field(i)
		mapFormField(typeField, structField, form, formfile, errors)
	}
}

func mapFormField(typeField reflect.StructField,
	structField reflect.Value, form map[string][]string,
	formfile map[string][]*multipart.FileHeader, errors Errors) {

	if !structField.CanSet() {
		return
	}

	if typeField.Type.Kind() == reflect.Ptr && typeField.Anonymous {
		structField.Set(reflect.New(typeField.Type.Elem()))
		mapForm(structField.Elem(), form, formfile, errors)
		if reflect.DeepEqual(structField.Elem().Interface(), reflect.Zero(structField.Elem().Type()).Interface()) {
			structField.Set(reflect.Zero(structField.Type()))
		}
	} else if typeField.Type.Kind() == reflect.Struct {
		mapForm(structField, form, formfile, errors)
	} else if typeField.Type.Kind() == reflect.Slice {
		mapFormFieldSlice(typeField, structField, form, formfile, errors)
	} else {
		mapFormFieldValue(typeField, structField, form, formfile, errors)
	}
}

func mapFormFieldSlice(typeField reflect.StructField,
	structField reflect.Value, form map[string][]string,
	formfile map[string][]*multipart.FileHeader, errors Errors) {

	if reflect.TypeOf((*multipart.FileHeader)(nil)) == structField.Type().Elem() {
		mapFormFieldMultipart(typeField, structField, formfile, errors)
	} else if structField.Type().Elem().Kind() == reflect.Struct {
		// TODO: element in slice is struct, iterate all and call mapForm
	} else {
		mapFormFieldSliceBuiltin(typeField, structField, form, errors)
	}
}

func mapFormFieldSliceBuiltin(typeField reflect.StructField,
        structField reflect.Value, form map[string][]string, errors Errors) {

	inputFieldName := typeField.Tag.Get("form")
	if inputFieldName == "" {
		return
	}

	inputValue, exists := form[inputFieldName]
	if !exists {
		return
	}

	numElems := len(inputValue)
	if numElems > 0 {
		sliceOf := structField.Type().Elem().Kind()
		slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
		for i := 0; i < numElems; i++ {
			setWithProperType(sliceOf, inputValue[i], slice.Index(i), inputFieldName, errors)
		}
		structField.Set(slice)
	}
}

func mapFormFieldValue(typeField reflect.StructField,
	structField reflect.Value, form map[string][]string,
	formfile map[string][]*multipart.FileHeader, errors Errors) {

	// handle multipart separately
	if structField.Type() == reflect.TypeOf((*multipart.FileHeader)(nil)) {
		mapFormFieldMultipart(typeField, structField, formfile, errors)
		return
	}

	inputFieldName := typeField.Tag.Get("form")
	if inputFieldName == "" {
		return
	}

	inputValue, exists := form[inputFieldName]
	if !exists {
		return
	}

	setWithProperType(typeField.Type.Kind(), inputValue[0], structField, inputFieldName, errors)
}

func mapFormFieldMultipart(typeField reflect.StructField,
	structField reflect.Value,
	formfile map[string][]*multipart.FileHeader, errors Errors) {

	inputFieldName := typeField.Tag.Get("form")
	if inputFieldName == "" {
		return
	}

	inputFile, exists := formfile[inputFieldName]
	if !exists {
		return
	}
	fhType := reflect.TypeOf((*multipart.FileHeader)(nil))
	numElems := len(inputFile)
	if structField.Kind() == reflect.Slice && numElems > 0 && structField.Type().Elem() == fhType {
		slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
		for i := 0; i < numElems; i++ {
			slice.Index(i).Set(reflect.ValueOf(inputFile[i]))
		}
		structField.Set(slice)
	} else if structField.Type() == fhType {
		structField.Set(reflect.ValueOf(inputFile[0]))
	}
}
