package calculate

import "errors"

// Тут функция для обработки ошибок
var (
	ErrInvalidExpression = errors.New("invalid expression")
	ErrDivisionByZero    = errors.New("division by zero")
	ErrEmptyExpression   = errors.New("empty expression")
)
