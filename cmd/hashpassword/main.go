// Command hashpassword prints the password hash an ADMIN_ACCOUNTS entry needs
// (see config.AdminConfig): it reads the password from stdin and prints
// base64 of its hash, ready for `email:<output>`.
//
//	echo -n 'the password' | go run ./cmd/hashpassword
package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"vn_startup_core_backend_go/internal/auth"
)

func main() {
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		fmt.Fprintln(os.Stderr, "hashpassword: read the password from stdin")
		os.Exit(1)
	}
	password := strings.TrimRight(line, "\r\n")
	if len(password) < 8 {
		fmt.Fprintln(os.Stderr, "hashpassword: the password must be at least 8 characters")
		os.Exit(1)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hashpassword:", err)
		os.Exit(1)
	}
	fmt.Println(base64.StdEncoding.EncodeToString([]byte(hash)))
}
