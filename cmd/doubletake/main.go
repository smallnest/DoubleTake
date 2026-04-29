package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
)

const (
	defaultPort = 8127
	usage       = `Usage: doubletake [options]

Options:
  --role string   Role mode: judge or player (required)
  --port int      Server port (default 8127)
  --addr string   Server address host:port (required for player mode)
  --stealth       Enable stealth mode (simplified output, no game markers)

Examples:
  doubletake --role judge
  doubletake --role player --addr 192.168.1.100:8127
  doubletake --role player --stealth
`
)

func run(stdout, stderr io.Writer, stdin io.Reader, args []string) int {
	var role string
	var port int
	var stealth bool
	var addr string

	i := 1 // skip program name
	for i < len(args) {
		switch args[i] {
		case "--role":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "Error: --role requires a value")
				fmt.Fprint(stderr, usage)
				return 1
			}
			role = args[i+1]
			i += 2
		case "--port":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "Error: --port requires a value")
				fmt.Fprint(stderr, usage)
				return 1
			}
			_, err := fmt.Sscanf(args[i+1], "%d", &port)
			if err != nil {
				fmt.Fprintf(stderr, "Error: invalid port value: %s\n", args[i+1])
				fmt.Fprint(stderr, usage)
				return 1
			}
			i += 2
		case "--addr":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "Error: --addr requires a value")
				fmt.Fprint(stderr, usage)
				return 1
			}
			addr = args[i+1]
			i += 2
		case "--stealth":
			stealth = true
			i++
		case "--help", "-h":
			fmt.Fprint(stdout, usage)
			return 0
		default:
			fmt.Fprintf(stderr, "Error: unknown option: %s\n", args[i])
			fmt.Fprint(stderr, usage)
			return 1
		}
	}

	if role == "" {
		fmt.Fprint(stdout, usage)
		return 0
	}

	if role != "judge" && role != "player" {
		fmt.Fprintf(stderr, "Error: invalid role '%s', must be judge or player\n", role)
		fmt.Fprint(stderr, usage)
		return 1
	}

	if port == 0 {
		port = defaultPort
	}

	switch role {
	case "judge":
		RunJudge(stdout, stdin, strconv.Itoa(port), stealth)
		return 0
	case "player":
		if addr == "" {
			fmt.Fprintln(stderr, "Error: --addr is required for player mode")
			fmt.Fprint(stderr, usage)
			return 1
		}
		return RunPlayer(stdout, stdin, stealth, addr)
	}

	return 0
}

func main() {
	os.Exit(run(os.Stdout, os.Stderr, os.Stdin, os.Args))
}
