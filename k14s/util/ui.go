package util

import (
	"log"

	. "github.com/cppforlife/go-cli-ui/ui/table"
)

type LoggingUI struct {
}

func (u *LoggingUI) ErrorLinef(pattern string, args ...interface{}) {
	log.Printf("Error: "+pattern+"\n", args...)
}

func (u *LoggingUI) PrintLinef(pattern string, args ...interface{}) {
	log.Printf("Error: "+pattern+"\n", args...)
}

func (u *LoggingUI) BeginLinef(pattern string, args ...interface{}) {
	log.Printf("--> "+pattern, args...)
}

func (u *LoggingUI) EndLinef(pattern string, args ...interface{}) {
	log.Printf(pattern+"<---\n", args...)
}

func (u *LoggingUI) PrintBlock(block []byte) {
	log.Println(string(block))
}

func (u *LoggingUI) PrintErrorBlock(msg string) {
	log.Println(msg)
}

func (u *LoggingUI) PrintTable(Table) {
	log.Println("some table")
}

func (u *LoggingUI) AskForText(label string) (string, error) {
	return "", nil
}

func (u *LoggingUI) AskForChoice(label string, options []string) (int, error) {
	return 0, nil
}

func (u *LoggingUI) AskForPassword(label string) (string, error) {
	return "", nil
}

// AskForConfirmation returns error if user doesnt want to continue
func (u *LoggingUI) AskForConfirmation() error {
	return nil
}

func (u *LoggingUI) IsInteractive() bool {
	return false
}

func (u *LoggingUI) Flush() {
	log.Println("")
}
