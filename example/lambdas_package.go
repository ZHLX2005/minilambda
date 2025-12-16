package example

import (
	"context"
	"fmt"
	"github.com/ZHLX2005/minilambda/core"
	"github.com/ZHLX2005/minilambda/registry"
	"strconv"
	"strings"
	"time"
)

// RegisterExampleLambdas 注册示例lambda函数
func RegisterExampleLambdas() {
	// 注册字符串处理lambda
	registry.RegisterLambda("string_upper", stringToUpper)
	registry.RegisterLambda("string_lower", stringToLower)
	registry.RegisterLambda("string_reverse", stringReverse)
	registry.RegisterLambda("string_length", stringLength)

	// 注册数学计算lambda
	registry.RegisterLambda("math_double", mathDouble)
	registry.RegisterLambda("math_square", mathSquare)
	registry.RegisterLambda("math_factorial", mathFactorial)

	// 注册数据转换lambda
	registry.RegisterLambda("int_to_string", intToString)
	registry.RegisterLambda("string_to_int", stringToInt)

	// 注册带选项的lambda
	registry.RegisterLambda("greeting_with_options", greetingWithOptions,
		core.WithTimeout(5*time.Second),
		core.WithEnableMetrics(true),
	)
}

// 字符串处理函数
func stringToUpper(ctx context.Context, input string) (string, error) {
	return strings.ToUpper(input), nil
}

func stringToLower(ctx context.Context, input string) (string, error) {
	return strings.ToLower(input), nil
}

func stringReverse(ctx context.Context, input string) (string, error) {
	runes := []rune(input)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes), nil
}

func stringLength(ctx context.Context, input string) (int, error) {
	return len(input), nil
}

// 数学计算函数
func mathDouble(ctx context.Context, input int) (int, error) {
	return input * 2, nil
}

func mathSquare(ctx context.Context, input int) (int, error) {
	return input * input, nil
}

func mathFactorial(ctx context.Context, input int) (int, error) {
	if input < 0 {
		return 0, fmt.Errorf("factorial is not defined for negative numbers")
	}
	if input == 0 || input == 1 {
		return 1, nil
	}

	result := 1
	for i := 2; i <= input; i++ {
		result *= i
	}
	return result, nil
}

// 数据转换函数
func intToString(ctx context.Context, input int) (string, error) {
	return strconv.Itoa(input), nil
}

func stringToInt(ctx context.Context, input string) (int, error) {
	return strconv.Atoi(input)
}

// 带选项的问候函数
func greetingWithOptions(ctx context.Context, input string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(100 * time.Millisecond):
		// 模拟一些处理时间
	}

	return fmt.Sprintf("Hello, %s! Welcome to MiniLambda!", input), nil
}

// 复杂数据结构示例
type Person struct {
	Name string
	Age  int
}

type PersonGreeting struct {
	Message string
	IsValid bool
}

func init() {
	// 注册复杂lambda
	registry.RegisterLambda("validate_person", validatePerson)
	registry.RegisterLambda("create_greeting", createGreeting)
}

func validatePerson(ctx context.Context, input Person) (PersonGreeting, error) {
	if input.Name == "" {
		return PersonGreeting{IsValid: false}, nil
	}
	if input.Age < 0 || input.Age > 150 {
		return PersonGreeting{IsValid: false}, nil
	}

	return PersonGreeting{
		Message: fmt.Sprintf("Valid person: %s, age %d", input.Name, input.Age),
		IsValid: true,
	}, nil
}

func createGreeting(ctx context.Context, input Person) (PersonGreeting, error) {
	if input.Name == "" {
		return PersonGreeting{IsValid: false}, fmt.Errorf("name cannot be empty")
	}

	var message string
	if input.Age < 18 {
		message = fmt.Sprintf("Hi %s! You're young and full of potential!", input.Name)
	} else if input.Age < 65 {
		message = fmt.Sprintf("Hello %s! You're in the prime of your life!", input.Name)
	} else {
		message = fmt.Sprintf("Greetings %s! You have a wealth of experience!", input.Name)
	}

	return PersonGreeting{
		Message: message,
		IsValid: true,
	}, nil
}
