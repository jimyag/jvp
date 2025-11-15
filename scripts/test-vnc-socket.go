package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <vnc-socket-path>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s /var/lib/jvp/qemu/i-xxx.vnc\n", os.Args[0])
		os.Exit(1)
	}

	socketPath := os.Args[1]

	fmt.Printf("Testing VNC socket: %s\n", socketPath)

	// 1. 检查文件是否存在
	stat, err := os.Stat(socketPath)
	if err != nil {
		fmt.Printf("✗ Socket file error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Socket file exists\n")
	fmt.Printf("  Mode: %v\n", stat.Mode())
	fmt.Printf("  Size: %d bytes\n", stat.Size())

	// 2. 尝试连接
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		fmt.Printf("✗ Failed to connect to socket: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("✓ Successfully connected to socket\n")

	// 3. 尝试读取 VNC 握手
	// VNC 服务器应该发送版本字符串，例如 "RFB 003.008\n"
	buffer := make([]byte, 12)
	n, err := conn.Read(buffer)
	if err != nil {
		if err == io.EOF {
			fmt.Printf("✗ Connection closed immediately (no data)\n")
		} else {
			fmt.Printf("✗ Read error: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("✓ Received %d bytes from VNC server\n", n)
	fmt.Printf("  Data: %q\n", string(buffer[:n]))

	if n >= 11 && string(buffer[:3]) == "RFB" {
		fmt.Printf("✓ Valid VNC handshake detected\n")
		fmt.Printf("  VNC Protocol version: %s\n", string(buffer[:n]))
	} else {
		fmt.Printf("? Unexpected data (not standard VNC)\n")
	}

	fmt.Println("\n=== Test completed successfully ===")
}
