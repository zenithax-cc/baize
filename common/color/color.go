package color

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
)

func Red(s string) string {
	return red + s + reset
}

func Green(s string) string {
	return green + s + reset
}

func Yellow(s string) string {
	return yellow + s + reset
}
