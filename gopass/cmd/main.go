// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 sudoforge <sudoforge.com>

package main

import (
	"fmt"
	"os"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/gopass"
)

func main() {
	helper, err := gopass.New()
	if err != nil {
		fmt.Printf("unable to use helper 'gopass': %v", err)
		os.Exit(1)
	}

	credentials.Serve(helper)
}
