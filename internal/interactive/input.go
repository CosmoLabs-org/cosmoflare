package interactive

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// InputReader abstracts reading user input for testability.
type InputReader interface {
	ReadLine() (string, error)
}

// stdinReader reads lines from os.Stdin.
type stdinReader struct{}

func (r *stdinReader) ReadLine() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	return strings.TrimSpace(line), err
}

// DefaultInput returns the standard stdin-based InputReader.
func DefaultInput() InputReader {
	return &stdinReader{}
}

// PromptWithReader prompts the user with a default value, reading from the given InputReader.
func PromptWithReader(prompt, defaultValue string, reader InputReader) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	response, err := reader.ReadLine()
	if err != nil {
		return "", err
	}

	if response == "" {
		return defaultValue, nil
	}
	return response, nil
}

// SelectFromListWithReader prompts the user to select from a list, reading from the given InputReader.
func SelectFromListWithReader(prompt string, options []string, defaultIndex int, reader InputReader) (int, error) {
	fmt.Println(prompt)
	fmt.Println()

	for i, option := range options {
		if i == defaultIndex {
			fmt.Printf("► [%d] %s\n", i+1, option)
		} else {
			fmt.Printf("  [%d] %s\n", i+1, option)
		}
	}
	fmt.Println()

	for {
		if defaultIndex >= 0 {
			fmt.Printf("Select option [%d]: ", defaultIndex+1)
		} else {
			fmt.Print("Select option: ")
		}

		response, err := reader.ReadLine()
		if err != nil {
			return -1, err
		}

		if response == "" && defaultIndex >= 0 {
			return defaultIndex, nil
		}

		index, err := strconv.Atoi(response)
		if err != nil || index < 1 || index > len(options) {
			fmt.Printf("  Please enter a number between 1 and %d\n", len(options))
			continue
		}

		return index - 1, nil
	}
}

// ConfirmWithReader prompts for yes/no confirmation, reading from the given InputReader.
func ConfirmWithReader(prompt string, defaultYes bool, reader InputReader) bool {
	defaultText := "Y/n"
	if !defaultYes {
		defaultText = "y/N"
	}

	fmt.Printf("%s [%s]: ", prompt, defaultText)

	response, err := reader.ReadLine()
	if err != nil {
		return defaultYes
	}
	response = strings.ToLower(response)

	if response == "" {
		return defaultYes
	}

	return response == "y" || response == "yes"
}
