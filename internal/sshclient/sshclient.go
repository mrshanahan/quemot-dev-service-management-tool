package sshclient

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

func CreateSshClient(addr string, user string, keyPath string, keyPassphrase string) (*ssh.Client, error) {
	// Significant components of this taken from example in docs:
	// https://pkg.go.dev/golang.org/x/crypto@v0.36.0/ssh#example-PublicKeys
	// https://pkg.go.dev/golang.org/x/crypto@v0.36.0/ssh#Dial

	// var hostKey ssh.PublicKey

	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key %s: %w", keyPath, err)
	}

	var signer ssh.Signer
	if keyPassphrase != "" {
		slog.Debug("parsing private key with provided passphrase", "key-path", keyPath)
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(keyPassphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(key)
	}

	if err != nil {
		return nil, fmt.Errorf("unable to parse private key - ensure the correct passphrase is provided: %w", err)
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		// HostKeyCallback: ssh.FixedHostKey(hostKey),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if !strings.Contains(addr, ":") {
		addr = fmt.Sprintf("%s:22", addr)
	}
	slog.Info("dialing ssh server", "addr", addr, "config", config)

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to remove server: %w", err)
	}

	slog.Info("successfully dialed ssh server", "addr", addr, "config", config)
	return client, nil
}
