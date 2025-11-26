package cmd

import (
	"fmt"
	"log"
	"strings"
	"syscall"

	"golang.org/x/term"
)

func readMFACode() string {
	fmt.Print("Enter MFA code: ")
	var code string
	var char byte
	buf := make([]byte, 1)

	oldState, err := term.MakeRaw(int(syscall.Stdin))
	if err != nil {
		log.Fatalf("❌ Failed to set terminal mode: %v", err)
	}
	defer term.Restore(int(syscall.Stdin), oldState)

	for {
		_, err := syscall.Read(syscall.Stdin, buf)
		if err != nil {
			log.Fatalf("❌ Failed to read input: %v", err)
		}
		char = buf[0]

		if char == 13 || char == 10 { // Enter
			fmt.Print("\r\n")
			break
		} else if char == 127 || char == 8 { // Backspace
			if len(code) > 0 {
				code = code[:len(code)-1]
				fmt.Print("\b \b")
			}
		} else if char >= 32 && char <= 126 { // Printable characters
			code += string(char)
			fmt.Print("*")
		}
	}

	return strings.TrimSpace(code)
}
