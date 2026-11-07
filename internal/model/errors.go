package model

import "errors"

var (
	ErrNotFound  = errors.New("记录不存在")
	ErrShelfFull = errors.New("书架已满")
)
