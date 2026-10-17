package db

import "errors"

var (
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrSessionNotFound     = errors.New("session not found")
	ErrSharedEventNotFound = errors.New("shared event not found")
	ErrTagAlreadyExists    = errors.New("tag already exists")
	ErrTagNotFound         = errors.New("tag not found")
	ErrLastTag             = errors.New("at least one tag is required")
	ErrDefaultTagNotFound  = errors.New("default tag not found")
	ErrHouseholdNotFound   = errors.New("household not found")
	ErrAlreadyInHousehold  = errors.New("user already belongs to a household")
)
