// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

// This file describes the code of the built-in command used by lsftp.
// It is quite big in that relationship. Maybe it will be separated or repaired soon.

package sftp

import (
	"fmt"
	"os"
	"path/filepath"
)

// pwd ...
func (r *RunSftp) pwd() {
	exit := make(chan bool)

	for s, c := range r.Client {
		server, client := s, c

		go func() {
			defer func() { exit <- true }()

			// get writer
			client.Output.Create(server)
			w := client.Output.NewWriter()

			// get current directory
			pwd, _ := client.Connect.Getwd()

			if len(client.Pwd) != 0 {
				if filepath.IsAbs(client.Pwd) {
					pwd = client.Pwd
				} else {
					pwd = filepath.Join(pwd, client.Pwd)
				}
			}

			fmt.Fprintf(w, "%s\n", pwd)
		}()
	}

	for range r.Client {
		<-exit
	}
}

// lpwd ...
func (r *RunSftp) lpwd() {
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	fmt.Println(pwd)
}
