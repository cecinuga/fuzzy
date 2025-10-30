package flaggy

import (
	"fmt"
	"fuzzy/internal/utils"
	"os"
)

type Spec[T any] struct { 
	usage 		string
	_default 	string
	validator 	utils.Matcher[T]
}

func (o *Spec[any]) Help(){
	fmt.Println(o.usage)
}

func (o *Spec[any]) Validate(obj any){
	if !o.validator(obj){
		o.Help()
		os.Exit(1)
	}
}

type Specs map[string]any

func (o *Specs) String(key, usage, _default string, validator utils.Matcher[string]){
	spec := Spec[string]{usage, _default, validator}
	(*o)[key] = spec
}

func (o *Specs) Bool(key, usage, _default string, validator utils.Matcher[bool]){
	spec := Spec[bool]{usage, _default, validator}
	(*o)[key] = spec
}

// Metodi per recuperare gli spec con type assertion
func (o *Specs) GetStringSpec(key string) (Spec[string], bool) {
	if spec, exists := (*o)[key]; exists {
		if stringSpec, ok := spec.(Spec[string]); ok {
			return stringSpec, true
		}
	}
	return Spec[string]{}, false
}

func (o *Specs) GetBoolSpec(key string) (Spec[bool], bool) {
	if spec, exists := (*o)[key]; exists {
		if boolSpec, ok := spec.(Spec[bool]); ok {
			return boolSpec, true
		}
	}
	return Spec[bool]{}, false
}

func (o *Specs) Manual() {
	for _, val := range *o {
		spec := val.(Spec[any])

		fmt.Printf(spec.usage)
	}	
}

// Metodo generico per recuperare spec di qualsiasi tipo
func (o *Specs) GetSpec(key string) (any, bool) {
	spec, exists := (*o)[key]
	return spec, exists
}

type Option[T any] struct {
	value T
	spec *Spec[T]
}

type Options map[string]any

func (o Options) ParseFlags(specs Specs){
	// TODO: Implementare il parsing degli argomenti da os.Args
	// Usando gli specs per validare i flag
}