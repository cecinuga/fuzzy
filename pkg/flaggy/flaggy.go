package flaggy

import (
	"fmt"
	"fuzzy/internal/utils"
	"os"
)

type Spec[T any] struct { 
	usage 		string
	_default 	T
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

type Option[T any] struct {
	value T
	spec *Spec[T]
}

func (o *Option[T]) Value() T {
	// Usa reflection per controllare se il valore è zero
	var zero T
	if any(o.value) != any(zero) {
		return o.value
	}
	return o.spec._default
}

func (o *Option[T]) Help(){
	fmt.Println(o.spec.usage)
}

func (o *Option[T]) validate(){
	o.spec.Validate(o.value)
}

type Options map[string]any

func (o *Options) getStringOption(key string) (*Option[string], bool) {
	if opt, exists := (*o)[key]; exists {
		if stringOpt, ok := opt.(Option[string]); ok {
			return &stringOpt, true
		}
	}
	return nil, false
}

func (o *Options) getBoolOption(key string) (*Option[bool], bool) {
	if opt, exists := (*o)[key]; exists {
		if boolOpt, ok := opt.(Option[bool]); ok {
			return &boolOpt, true
		}
	}
	return nil, false
}

func (o *Options) String(key, usage, _default string, validator utils.Matcher[string]) (*Option[string], bool){
	option := Option[string]{spec: &Spec[string]{usage, _default, validator}}
	(*o)[key] = option	

	return o.getStringOption(key)
}

func (o *Options) Bool(key, usage string, _default bool, validator utils.Matcher[bool]) (*Option[bool], bool){
	option := Option[bool]{spec: &Spec[bool]{usage, _default, validator}}
	(*o)[key] = option	

	return o.getBoolOption(key)
}


func (o Options) ParseFlags(){
	// TODO: Implementare il parsing degli argomenti da os.Args
	// Usando gli specs per validare i flag
}