package credentials

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type credentialsGetResponse struct {
	Username string
	Secret   string
}

// Serve initializes the credentials helper and parses the action argument.
func Serve(helper Helper) {
	if err := handleCommand(helper); err != nil {
		fmt.Fprintf(os.Stdout, "%v\n", err)
		os.Exit(1)
	}
}

func handleCommand(helper Helper) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("Usage: %s <store|get|erase>", os.Args[0])
	}

	switch os.Args[1] {
	case "store":
		return store(helper, os.Stdin)
	case "get":
		return get(helper, os.Stdin, os.Stdout)
	case "erase":
		return erase(helper, os.Stdin)
	}
	return fmt.Errorf("Usage: %s <store|get|erase>", os.Args[0])
}

func store(helper Helper, reader io.Reader) error {
	scanner := bufio.NewScanner(reader)

	buffer := new(bytes.Buffer)
	for scanner.Scan() {
		buffer.Write(scanner.Bytes())
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return err
	}

	var creds Credentials
	if err := json.NewDecoder(buffer).Decode(&creds); err != nil {
		return err
	}

	return helper.Add(&creds)
}

func get(helper Helper, reader io.Reader, writer io.Writer) error {
	scanner := bufio.NewScanner(reader)

	buffer := new(bytes.Buffer)
	for scanner.Scan() {
		buffer.Write(scanner.Bytes())
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return err
	}

	serverURL := strings.TrimSpace(buffer.String())

	username, secret, err := helper.Get(serverURL)
	if err != nil {
		return err
	}

	resp := credentialsGetResponse{
		Username: username,
		Secret:   secret,
	}

	buffer.Reset()
	if err := json.NewEncoder(buffer).Encode(resp); err != nil {
		return err
	}

	fmt.Fprint(writer, buffer.String())
	return nil
}

func erase(helper Helper, reader io.Reader) error {
	scanner := bufio.NewScanner(reader)

	buffer := new(bytes.Buffer)
	for scanner.Scan() {
		buffer.Write(scanner.Bytes())
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return err
	}

	serverURL := strings.TrimSpace(buffer.String())

	return helper.Delete(serverURL)
}
