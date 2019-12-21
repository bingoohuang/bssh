// Copyright (c) 2019 Blacknon. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.

package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/blacknon/lssh/misc"

	"github.com/bingoohuang/gou/pbe"
	sshlib "github.com/blacknon/go-sshlib"
	"github.com/blacknon/lssh/common"
	"golang.org/x/crypto/ssh"
)

// CreateAuthMethodMap Create ssh.AuthMethod, into r.AuthMethodMap.
func (r *Run) CreateAuthMethodMap() {
	srvs := r.ServerList

	for _, server := range r.ServerList {
		proxySrvs, _ := getProxyRoute(server, r.Conf)

		for _, proxySrv := range proxySrvs {
			if proxySrv.Type == misc.SSH {
				srvs = append(srvs, proxySrv.Name)
			}
		}
	}

	srvs = common.GetUniqueSlice(srvs)

	// Init r.AuthMethodMap
	r.authMethodMap = map[AuthKey][]ssh.AuthMethod{}
	r.serverAuthMethodMap = map[string][]ssh.AuthMethod{}

	for _, server := range srvs {
		// get server config
		config := r.Conf.Server[server]

		// Password
		if config.Pass != "" {
			r.registerAuthMapPassword(server, config.Pass)
		}

		// Multiple Password
		if len(config.Passes) > 0 {
			for _, pass := range config.Passes {
				r.registerAuthMapPassword(server, pass)
			}
		}

		// PublicKey
		if config.Key != "" {
			err := r.registerAuthMapPublicKey(server, config.Key, config.KeyPass)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}

		// Multiple PublicKeys
		if len(config.Keys) > 0 {
			for _, key := range config.Keys {
				//
				pair := strings.SplitN(key, "::", 2)
				keyName := pair[0]
				keyPass := ""

				//
				if len(pair) > 1 {
					keyPass = pair[1]
				}

				//
				err := r.registerAuthMapPublicKey(server, keyName, keyPass)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue
				}
			}
		}

		// Public Key Command
		if config.KeyCommand != "" {
			// TODO(blacknon): keyCommandの追加
			err := r.registerAuthMapPublicKeyCommand(server, config.KeyCommand, config.KeyCommandPass)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}

		// Certificate
		if config.Cert != "" {
			keySigner, err := sshlib.CreateSignerPublicKeyPrompt(config.CertKey, config.CertKeyPass)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}

			err = r.registerAuthMapCertificate(server, config.Cert, keySigner)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
		}

		// PKCS11
		if config.PKCS11Use {
			err := r.registerAuthMapPKCS11(server, config.PKCS11Provider, config.PKCS11PIN)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
	}
}

// SetupSSHAgent setup SSH agent.
func (r *Run) SetupSSHAgent() {
	// Connect ssh-agent
	r.agent = sshlib.ConnectSshAgent()
}

// registerAuthMapPassword ...
func (r *Run) registerAuthMapPassword(server, password string) {
	password = decodePassword(password)

	authKey := AuthKey{AuthKeyPassword, password}
	if _, ok := r.authMethodMap[authKey]; !ok {
		authMethod := sshlib.CreateAuthMethodPassword(password)

		// Register AuthMethod to authMethodMap
		r.authMethodMap[authKey] = append(r.authMethodMap[authKey], authMethod)
	}

	// Register AuthMethod to serverAuthMethodMap from authMethodMap
	r.serverAuthMethodMap[server] = append(r.serverAuthMethodMap[server], r.authMethodMap[authKey]...)
}

func decodePassword(password string) string {
	if pwd, err := pbe.Ebp(password); err != nil {
		panic(err)
	} else {
		return pwd
	}
}

//
func (r *Run) registerAuthMapPublicKey(server, key, password string) (err error) {
	authKey := AuthKey{AuthKeyKey, key}

	if _, ok := r.authMethodMap[authKey]; !ok {
		// Create signer with key input
		signer, err := sshlib.CreateSignerPublicKeyPrompt(key, password)
		if err != nil {
			return err
		}

		// Create AuthMethod
		authMethod := ssh.PublicKeys(signer)

		// Register AuthMethod to authMethodMap
		r.authMethodMap[authKey] = append(r.authMethodMap[authKey], authMethod)
	}

	// Register AuthMethod to serverAuthMethodMap from authMethodMap
	r.serverAuthMethodMap[server] = append(r.serverAuthMethodMap[server], r.authMethodMap[authKey]...)

	return
}

//
func (r *Run) registerAuthMapPublicKeyCommand(server, command, password string) (err error) {
	authKey := AuthKey{AuthKeyKey, command}

	if _, ok := r.authMethodMap[authKey]; !ok {
		// Run key command
		cmd := exec.Command("sh", "-c", command)
		keyData, err := cmd.Output()

		if err != nil {
			return err
		}

		// Create signer
		signer, err := sshlib.CreateSignerPublicKeyData(keyData, password)
		if err != nil {
			return err
		}

		// Create AuthMethod
		authMethod := ssh.PublicKeys(signer)

		// Register AuthMethod to authMethodMap
		r.authMethodMap[authKey] = append(r.authMethodMap[authKey], authMethod)
	}

	// Register AuthMethod to serverAuthMethodMap from authMethodMap
	r.serverAuthMethodMap[server] = append(r.serverAuthMethodMap[server], r.authMethodMap[authKey]...)

	return
}

//
func (r *Run) registerAuthMapCertificate(server, cert string, signer ssh.Signer) (err error) {
	authKey := AuthKey{AuthKeyCert, cert}

	if _, ok := r.authMethodMap[authKey]; !ok {
		authMethod, err := sshlib.CreateAuthMethodCertificate(cert, signer)
		if err != nil {
			return err
		}

		// Register AuthMethod to authMethodMap
		r.authMethodMap[authKey] = append(r.authMethodMap[authKey], authMethod)
	}

	// Register AuthMethod to serverAuthMethodMap from authMethodMap
	r.serverAuthMethodMap[server] = append(r.serverAuthMethodMap[server], r.authMethodMap[authKey]...)

	return
}

func (r *Run) registerAuthMapPKCS11(server, provider, pin string) (err error) {
	authKey := AuthKey{AuthKeyPkcs11, provider}
	if _, ok := r.authMethodMap[authKey]; !ok {
		// Create Signer with key input
		signers, err := sshlib.CreateSignerPKCS11Prompt(provider, pin)
		if err != nil {
			return err
		}

		for _, signer := range signers {
			// Create AuthMethod
			authMethod := ssh.PublicKeys(signer)

			// Register AuthMethod to AuthMethodMap
			r.authMethodMap[authKey] = append(r.authMethodMap[authKey], authMethod)
		}
	}

	// Register AuthMethod to serverAuthMethodMap from authMethodMap
	r.serverAuthMethodMap[server] = append(r.serverAuthMethodMap[server], r.authMethodMap[authKey]...)

	return
}

// registerAuthMapKeyCmd is exec keycmd, and register kyecmd result publickey to AuthMap.
// func registerAuthMapKeyCmd() () {}

// execKeyCommand
// func execKeyCmd() {}
