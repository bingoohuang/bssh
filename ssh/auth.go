package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bingoohuang/bssh/common"
	"github.com/bingoohuang/bssh/misc"
	"github.com/bingoohuang/bssh/sshlib"
	"github.com/bingoohuang/ngg/ss"
	"github.com/manifoldco/promptui"
	"golang.org/x/crypto/ssh"
)

// CreateAuthMethodMap Create ssh.AuthMethod, into r.AuthMethodMap.
func (r *Run) CreateAuthMethodMap() {
	srvs := r.getSSHServers()

	// Init r.AuthMethodMap
	r.authMethodMap = map[AuthKey][]ssh.AuthMethod{}
	r.serverAuthMethodMap = map[string][]ssh.AuthMethod{}

	defer r.autoEncryptPwd()

	for _, server := range srvs {
		r.createAuthMethodMapForServer(server)
	}
}

func (r *Run) getSSHServers() []string {
	srvs := r.ServerList

	for _, server := range r.ServerList {
		proxySrvs, _ := getProxyRoute(server, r.Conf)

		for _, proxySrv := range proxySrvs {
			if proxySrv.Type == misc.SSH {
				srvs = append(srvs, proxySrv.Name)
			}
		}
	}

	return common.GetUniqueSlice(srvs)
}

// SetupSSHAgent setup SSH agent.
func (r *Run) SetupSSHAgent() {
	// Connect ssh-agent
	r.agent = sshlib.ConnectSshAgent()
}

// registerAuthMapPassword ...
func (r *Run) registerAuthMapPassword(serverID, password, rawTemplLine string) {
	if password == "" {
		return
	}

	password = r.decodePassword(password, rawTemplLine)

	authKey := AuthKey{AuthKeyPassword, password}
	if _, ok := r.authMethodMap[authKey]; !ok {
		authMethod := sshlib.CreateAuthMethodPassword(password)

		// Register AuthMethod to authMethodMap
		r.authMethodMap[authKey] = append(r.authMethodMap[authKey], authMethod)
	}

	// Register AuthMethod to serverAuthMethodMap from authMethodMap
	r.serverAuthMethodMap[serverID] = append(r.serverAuthMethodMap[serverID], r.authMethodMap[authKey]...)
}

func (r *Run) decodePassword(password, rawTemplLine string) string {
	if strings.EqualFold(password, "{Prompt}") {
		prompt := promptui.Prompt{
			Label:       "Password",
			HideEntered: true,
		}

		result, err := prompt.Run()
		if err != nil {
			panic(err)
		}

		if rawTemplLine != "" {
			r.updatePromptPwd(password, result, rawTemplLine)
		}

		return result
	}
	if pwd, err := ss.PbeDecode(password); err != nil {
		panic(err)
	} else {
		if os.Getenv("ECHO_PWD") == "1" {
			fmt.Println(pwd)
			os.Exit(0)
		}
		r.registerAutoEncryptPwd(password)
		return pwd
	}
}

func (r *Run) registerAuthMapPublicKey(serverID, key, password string) (err error) {
	if key == "" {
		return nil
	}

	authKey := AuthKey{Type: AuthKeyKey, Value: key}

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
	r.serverAuthMethodMap[serverID] = append(r.serverAuthMethodMap[serverID], r.authMethodMap[authKey]...)

	return
}

func (r *Run) registerAuthMapPublicKeyCommand(server, command, password string) error {
	if command == "" {
		return nil
	}

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

	return nil
}

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

//func (r *Run) registerAuthMapPKCS11(server, provider, pin string) (err error) {
//	authKey := AuthKey{AuthKeyPkcs11, provider}
//	if _, ok := r.authMethodMap[authKey]; !ok {
//		// Create Signer with key input
//		signers, err := sshlib.CreateSignerPKCS11(provider, pin)
//		if err != nil {
//			return err
//		}
//
//		for _, signer := range signers {
//			// Create AuthMethod
//			authMethod := ssh.PublicKeys(signer)
//
//			// Register AuthMethod to AuthMethodMap
//			r.authMethodMap[authKey] = append(r.authMethodMap[authKey], authMethod)
//		}
//	}
//
//	// Register AuthMethod to serverAuthMethodMap from authMethodMap
//	r.serverAuthMethodMap[server] = append(r.serverAuthMethodMap[server], r.authMethodMap[authKey]...)
//
//	return
//}

// registerAuthMapKeyCmd is exec keycmd, and register kyecmd result publickey to AuthMap.
// func registerAuthMapKeyCmd() () {}

// execKeyCommand
// func execKeyCmd() {}
