package main

import (
	"fmt"
	"io"
	"os"

	"github.com/smallnest/doubletake/client"
)

const (
	defaultPort = 8127
	usage       = `Usage: doubletake [options]

Options:
  --role string   Role mode: judge or player (required)
  --port int      Server port (default 8127)
  --stealth       Enable stealth mode (simplified output, no game markers)

Examples:
  doubletake --role judge
  doubletake --role player --port 9000
  doubletake --role player --stealth
`
)

func run(stdout, stderr io.Writer, args []string) int {
	var role string
	var port int
	var stealth bool

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

	disp := client.NewDisplay(stdout, stealth)
	disp.PrintStartup()
	disp.Info("0000", fmt.Sprintf("mode=%s port=%d", role, port))

	return 0
}

func main() {
	os.Exit(run(os.Stdout, os.Stderr, os.Args))
}
