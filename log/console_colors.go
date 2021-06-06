package log

// ConsoleColorsType has a set of methods that return the color directive to change console color
type ConsoleColorsType struct{}

// Red returns red directive
func (ConsoleColorsType) Red() string {
	return "\033[31m"
}

// Green returns green directive
func (ConsoleColorsType) Green() string {
	return "\033[32m"
}

// Yellow returns yellow directive
func (ConsoleColorsType) Yellow() string {
	return "\033[33m"
}

// Blue returns blue directive
func (ConsoleColorsType) Blue() string {
	return "\033[34m"
}

// Purple returns purple directive
func (ConsoleColorsType) Purple() string {
	return "\033[35m"
}

// Cyan returns cyan directive
func (ConsoleColorsType) Cyan() string {
	return "\033[36m"
}

// White returns white directive
func (ConsoleColorsType) White() string {
	return "\033[37m"
}

// Reset returns original color directive
func (ConsoleColorsType) Reset() string {
	return "\033[0m"
}

var (
	// ConsoleColors is a ConsoleColorsType singleton, which has a set of methods that return the color directive to change console color
	ConsoleColors = ConsoleColorsType{}
)
