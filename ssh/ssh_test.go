package ssh

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
	"testing"
)

func TestPemFile(t *testing.T) {
	// 读取私钥文件
	key, err := os.ReadFile("my.pem")
	if err != nil {
		fmt.Printf("无法读取私钥文件: %v\n", err)
		return
	}

	// 解析私钥
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		fmt.Printf("无法解析私钥: %v\n", err)
		return
	}

	// 配置 SSH 客户端
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接到远程服务器
	client, err := ssh.Dial("tcp", "5.11.16.15:22", config)
	if err != nil {
		fmt.Printf("无法连接到远程服务器: %v\n", err)
		return
	}
	defer client.Close()

	// 执行远程命令示例
	session, err := client.NewSession()
	if err != nil {
		fmt.Printf("无法创建会话: %v\n", err)
		return
	}
	defer session.Close()

	// 执行远程命令
	output, err := session.CombinedOutput("ls -l")
	if err != nil {
		fmt.Printf("无法执行命令: %v\n", err)
		return
	}

	// 输出命令执行结果
	fmt.Println(string(output))
}
