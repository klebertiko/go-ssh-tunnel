package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
)

type Endpoint struct {
	Host string
	Port int
}

func (endpoint *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

type SSHtunnel struct {
	Local  *Endpoint
	Server *Endpoint
	Remote *Endpoint

	Config *ssh.ClientConfig
}

func (tunnel *SSHtunnel) Start() error {
	listener, err := net.Listen("tcp", tunnel.Local.String())
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go tunnel.forward(conn)
	}
}

func (tunnel *SSHtunnel) forward(localConn net.Conn) {
	serverConn, err := ssh.Dial("tcp", tunnel.Server.String(), tunnel.Config)
	if err != nil {
		fmt.Printf("Server dial error: %s\n", err)
		return
	}

	remoteConn, err := serverConn.Dial("tcp", tunnel.Remote.String())
	if err != nil {
		fmt.Printf("Remote dial error: %s\n", err)
		return
	}

	copyConn := func(writer, reader net.Conn) {
		_, err := io.Copy(writer, reader)
		if err != nil {
			fmt.Printf("io.Copy error: %s", err)
		}
	}

	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)
}

func main() {
	user := flag.String("user", "", "SSH username\n\tEG.: user")
	pass := flag.String("pass", "", "SSH password\n\tEg.: 1234")

	local := flag.String("local", "", "Local address to bind remote service\n\tEg.: localhost")
	localPort := flag.Int("localPort", 0, "Local port to bind remote service\n\tEg.: 3000")

	sshServer := flag.String("sshServer", "", "SSH server address\n\tEg.: 10.1.1.2")

	remote := flag.String("remote", "", "Remote service address\n\tEg.: localhost")
	remotePort := flag.Int("remotePort", 0, "Remote service port\n\tEg.: 3000")

	flag.Parse()
	if *user == "" || *pass == "" || *local == "" || *localPort == 0 || *sshServer == "" || *remote == "" || *remotePort == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	sshConfig := &ssh.ClientConfig{
		User: *user,
		Auth: []ssh.AuthMethod{
			ssh.Password(*pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	localEndpoint := &Endpoint{
		Host: *local,
		Port: *localPort,
	}

	serverEndpoint := &Endpoint{
		Host: *sshServer,
		Port: 22,
	}

	remoteEndpoint := &Endpoint{
		Host: *remote,
		Port: *remotePort,
	}

	tunnel := &SSHtunnel{
		Config: sshConfig,
		Local:  localEndpoint,
		Server: serverEndpoint,
		Remote: remoteEndpoint,
	}

	tunnel.Start()
}
