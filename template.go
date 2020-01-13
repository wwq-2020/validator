package main

const tplStr = `
// Validate{{.Name|title}} Validate {{.Name|title}}
func Validate{{.Name|title}}(src {{.Name}}) error {
{{range $idx,$field := .Fields}}
	if {{$field.IsEmpty|raw}} {
		return {{$field.Error|raw}}
	}
{{end}}
	return nil
}


// Validate{{.Name|title}}Pointer Validate {{.Name|title}}Pointer
func Validate{{.Name|title}}Pointer(src *{{.Name}}) error {
	if src == nil {
		return errors.New("src can't be nil")
	}
{{range $idx,$field := .Fields}}
	if {{$field.IsEmpty|raw}} {
		return {{$field.Error|raw}}
	}
{{end}}
	return nil
}
`
