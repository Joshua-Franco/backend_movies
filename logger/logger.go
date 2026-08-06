package logger

import (
	"fmt"
	"time"

	"github.com/fatih/color"
)

var (
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	blue   = color.New(color.FgCyan).SprintFunc()
	white  = color.New(color.FgWhite).SprintFunc()
	bold   = color.New(color.Bold).SprintFunc()
	dim    = color.New(color.Faint).SprintFunc()
)

func timestamp() string {
	return time.Now().Format("15:04:05")
}

func Info(msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	fmt.Printf("%s %s %s\n", dim(timestamp()), blue("INFO"), formatted)
}

func Success(msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	fmt.Printf("%s %s %s\n", dim(timestamp()), green("OK"), formatted)
}

func Warn(msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	fmt.Printf("%s %s %s\n", dim(timestamp()), yellow("WARN"), formatted)
}

func Error(msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	fmt.Printf("%s %s %s\n", dim(timestamp()), red("ERROR"), formatted)
}

func Fatal(msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	fmt.Printf("\n%s %s %s\n\n", dim(timestamp()), red("FATAL"), formatted)
}

func Request(method, path string, status int) {
	statusStr := fmt.Sprintf("%d", status)
	switch {
	case status >= 200 && status < 300:
		statusStr = green(statusStr)
	case status >= 400 && status < 500:
		statusStr = yellow(statusStr)
	case status >= 500:
		statusStr = red(statusStr)
	}
	fmt.Printf("%s %s %s %s\n", dim(timestamp()), blue(method), white(path), statusStr)
}

func CacheHit(key string) {
	fmt.Printf("%s %s %s %s\n", dim(timestamp()), green("CACHE"), green("HIT"), dim(key))
}

func CacheMiss(key string) {
	fmt.Printf("%s %s %s %s\n", dim(timestamp()), yellow("CACHE"), yellow("MISS"), dim(key))
}

func Banner() {
	fmt.Println()
	fmt.Printf("  %s\n", bold("MOVIE NOVA Backend"))
	fmt.Printf("  %s\n", dim("Go + PostgreSQL + TMDB API"))
	fmt.Println()
}

func Separator() {
	fmt.Println(dim("─────────────────────────────────────────"))
}
