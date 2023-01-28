package main

import "fmt"

type ResizeErrors struct {
	Errors []*ResizeError `json:"errors"`
}

func NewResizeErrors() *ResizeErrors {
	return &ResizeErrors{
		Errors: make([]*ResizeError, 0),
	}
}

func (errs *ResizeErrors) Error() string {
	var str string
	for _, e := range errs.Errors {
		str += e.Error() + " "
	}
	return str
}

func (errs *ResizeErrors) Add(err *ResizeError) {
	errs.Errors = append(errs.Errors, err)
}

type ResizeError struct {
	Key string `json:"key"`
	Err error  `json:"error"`
}

func (e *ResizeError) Error() string {
	return fmt.Sprintf("%s: %s", e.Key, e.Err.Error())
}
