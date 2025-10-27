package config

import "fuzzy/internal/utils"

type Flag[T any] struct {
	Flag string
	Usage string
	Value T

	validators []utils.Matcher
}

func (f *Flag[T]) Check(){

}

func ParseFlags() (*Config) {
	config := Config{}

	return &config
}
