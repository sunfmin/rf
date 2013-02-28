package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"text/template"
	"unicode"
	"unicode/utf8"
)

// A Command is an implementation of a go command
// like go build or go fix.
type Command struct {
	// Run runs the command.
	// The args are the arguments after the command name.
	Run func(cmd *Command, args []string)

	// UsageLine is the one-line usage message.
	// The first word in the line is taken to be the command name.
	UsageLine string

	// Short is the short description shown in the 'go help' output.
	Short string

	// Long is the long message shown in the 'go help <this-command>' output.
	Long string

	// Flag is a set of flags specific to this command.
	Flag flag.FlagSet

	// CustomFlags indicates that the command will do its own
	// flag parsing.
	CustomFlags bool
}

// Name returns the command's name: the first word in the usage line.
func (c *Command) Name() string {
	name := c.UsageLine
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

func (c *Command) Usage() {
	fmt.Fprintf(os.Stderr, "usage: %s\n\n", c.UsageLine)
	fmt.Fprintf(os.Stderr, "%s\n", strings.TrimSpace(c.Long))
	os.Exit(2)
}

func (c *Command) Runnable() bool {
	return c.Run != nil
}

var commands = []*Command{
	cmdVacuum,
}

type Config struct {
	Packages       []string
	NoticePackages []string
	Entrances      []string
}

var config *Config

func main() {
	flag.Usage = usage
	flag.Parse()
	log.SetFlags(0)

	args := flag.Args()
	if len(args) < 1 {
		usage()
	}

	if args[0] == "help" {
		help(args[1:])
		return
	}

	configFile, err := os.Open(".rf")
	if err == nil {
		err = json.NewDecoder(configFile).Decode(&config)
		if err != nil {
			jsonErr := err.(*json.SyntaxError)
			errorfExit("rf: config file .rf parse error `%s` at %d\n", err, jsonErr.Offset)
		}
	}

	for _, cmd := range commands {
		if cmd.Name() == args[0] && cmd.Run != nil {
			cmd.Flag.Usage = func() { cmd.Usage() }
			if cmd.CustomFlags {
				args = args[1:]
			} else {
				cmd.Flag.Parse(args[1:])
				args = cmd.Flag.Args()
			}
			cmd.Run(cmd, args)
			exit()
			return
		}
	}

	errorfExit("rf: unknown subcommand %q\nRun 'rf help' for usage.\n", args[0])
}

var usageTemplate = `rf is a tool for refactoring Go source code.

Usage:

	rf command [arguments]

The commands are:
{{range .}}{{if .Runnable}}
    {{.Name | printf "%-11s"}} {{.Short}}{{end}}{{end}}

Use "rf help [command]" for more information about a command.

Additional help topics:
{{range .}}{{if not .Runnable}}
    {{.Name | printf "%-11s"}} {{.Short}}{{end}}{{end}}

Use "rf help [topic]" for more information about that topic.

`

var helpTemplate = `{{if .Runnable}}usage: rf {{.UsageLine}}

{{end}}{{.Long | trim}}
`

// tmpl executes the given template text on data, writing the result to w.
func tmpl(w io.Writer, text string, data interface{}) {
	t := template.New("top")
	t.Funcs(template.FuncMap{"trim": strings.TrimSpace, "capitalize": capitalize})
	template.Must(t.Parse(text))
	if err := t.Execute(w, data); err != nil {
		panic(err)
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToTitle(r)) + s[n:]
}

func printUsage(w io.Writer) {
	tmpl(w, usageTemplate, commands)
}

func usage() {
	printUsage(os.Stderr)
	os.Exit(2)
}

func errorfExit(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	setExitStatus(2)
	exit()
}

// help implements the 'help' command.
func help(args []string) {
	if len(args) == 0 {
		printUsage(os.Stdout)
		return
	}
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: rf help command\n\nToo many arguments given.\n")
		os.Exit(2) // failed at 'go help'
	}

	arg := args[0]

	for _, cmd := range commands {
		if cmd.Name() == arg {
			tmpl(os.Stdout, helpTemplate, cmd)
			// not exit 2: succeeded at 'go help cmd'.
			return
		}
	}

	fmt.Fprintf(os.Stderr, "Unknown help topic %#q.  Run 'rf help'.\n", arg)
	os.Exit(2) // failed at 'go help cmd'
}

var exitStatus = 0
var exitMu sync.Mutex

func setExitStatus(n int) {
	exitMu.Lock()
	if exitStatus < n {
		exitStatus = n
	}
	exitMu.Unlock()
}

var atexitFuncs []func()

func atexit(f func()) {
	atexitFuncs = append(atexitFuncs, f)
}

func exit() {
	for _, f := range atexitFuncs {
		f()
	}
	os.Exit(exitStatus)
}
