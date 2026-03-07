package main

import (
	"context"
	"efctl/pkg/container"
	"fmt"
)

func main() {
	client, err := container.NewClient()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	out, err := client.ExecCapture(context.Background(), "sui-playground", []string{"cat", "/root/.sui/.env.sui"})
	if err != nil {
		fmt.Println("Error executing cat:", err)
		return
	}
	fmt.Printf("Output Length: %d\n", len(out))
	fmt.Printf("Output:\n%s\n", out)
}
