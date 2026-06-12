package commandutil

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"

	"github.com/docker/hub-tool/internal/ansi"
	"github.com/docker/hub-tool/internal/errdef"
	"github.com/docker/hub-tool/internal/format"
	"github.com/docker/hub-tool/internal/format/tabwriter"
	"github.com/docker/hub-tool/internal/metrics"
	"github.com/docker/hub-tool/pkg/hub"
)

// CommandConfig contains the common Cobra command fields used by subcommands.
type CommandConfig struct {
	Use         string
	Aliases     []string
	Short       string
	Args        cobra.PositionalArgs
	Annotations map[string]string
	Parent      string
	Name        string
	RunE        func(*cobra.Command, []string) error
}

// SudoAnnotation marks commands that require authenticated Docker Hub access.
func SudoAnnotation() map[string]string {
	return map[string]string{"sudo": "true"}
}

// Column describes one printable table column.
type Column[T any] struct {
	Header string
	Value  func(T) (string, int)
}

// ListOptions contains the common --format and --all list flags.
type ListOptions struct {
	format.Option
	All bool
}

// AddListFlags adds the common list command flags.
func (opts *ListOptions) AddListFlags(cmd *cobra.Command, allHelp string) {
	cmd.Flags().BoolVar(&opts.All, "all", false, allHelp)
	opts.AddFormatFlag(cmd.Flags())
}

// TextColumn builds a string-valued table column.
func TextColumn[T any](header string, value func(T) string) Column[T] {
	return Column[T]{
		Header: header,
		Value: func(row T) (string, int) {
			s := value(row)
			return s, len(s)
		},
	}
}

// IntColumn builds an integer-valued table column.
func IntColumn[T any](header string, value func(T) int) Column[T] {
	return TextColumn(header, func(row T) string {
		return fmt.Sprintf("%d", value(row))
	})
}

// BoolColumn builds a boolean-valued table column.
func BoolColumn[T any](header string, value func(T) bool) Column[T] {
	return TextColumn(header, func(row T) string {
		return fmt.Sprintf("%v", value(row))
	})
}

// NewCommand builds a Cobra command with common flag/metric wiring.
func NewCommand(config CommandConfig) *cobra.Command {
	return &cobra.Command{
		Use:                   config.Use,
		Aliases:               config.Aliases,
		Short:                 config.Short,
		Args:                  config.Args,
		DisableFlagsInUseLine: true,
		Annotations:           config.Annotations,
		PreRun: func(cmd *cobra.Command, args []string) {
			metrics.Send(config.Parent, config.Name)
		},
		RunE: config.RunE,
	}
}

// NewForceCommand builds a command with the common -f/--force flag.
func NewForceCommand(config CommandConfig, force *bool, help string) *cobra.Command {
	cmd := NewCommand(config)
	cmd.Flags().BoolVarP(force, "force", "f", false, help)
	return cmd
}

// NewParentCommand builds a command namespace that only displays help itself.
func NewParentCommand(streams command.Streams, name, short string, annotations map[string]string) *cobra.Command {
	return &cobra.Command{
		Use:                   name,
		Short:                 short,
		Args:                  cli.NoArgs,
		DisableFlagsInUseLine: true,
		Annotations:           annotations,
		RunE:                  command.ShowHelp(streams.Err()),
	}
}

// PrintTable renders rows with the tabwriter style used by list commands.
func PrintTable[T any](out io.Writer, rows []T, columns []Column[T]) error {
	tw := tabwriter.New(out, "    ")
	for _, column := range columns {
		tw.Column(ansi.Header(column.Header), len(column.Header))
	}
	tw.Line()

	for _, row := range rows {
		for _, column := range columns {
			value, width := column.Value(row)
			tw.Column(value, width)
		}
		tw.Line()
	}
	return tw.Flush()
}

// PrettyTable adapts PrintTable to the formatter callback used by commands.
func PrettyTable[T any](columns []Column[T], total ...int) func(io.Writer, interface{}) error {
	return func(out io.Writer, values interface{}) error {
		rows := values.([]T)
		if err := PrintTable(out, rows, columns); err != nil {
			return err
		}
		if len(total) > 0 {
			PrintPaginationHint(out, len(rows), total[0])
		}
		return nil
	}
}

// UpdateAllElements applies the shared --all list behavior.
func UpdateAllElements(client *hub.Client, all bool) error {
	if !all {
		return nil
	}
	return client.Update(hub.WithAllElements())
}

// PrintPaginationHint prints the common list footer when only one page was fetched.
func PrintPaginationHint(out io.Writer, listed, total int) {
	if listed < total {
		_, _ = fmt.Fprintln(out, ansi.Info(fmt.Sprintf("%v/%v listed, use --all flag to show all", listed, total)))
	}
}

// ConfirmUnlessForced runs the common forced-delete confirmation flow.
func ConfirmUnlessForced(ctx context.Context, in io.Reader, force bool, validate func(string) error) error {
	if force {
		return nil
	}
	input, err := ReadConfirmation(ctx, in)
	if err != nil {
		return err
	}
	return validate(input)
}

// MatchConfirmation validates confirmation input against an expected value.
func MatchConfirmation(expected string, errf func(string) error) func(string) error {
	return func(input string) error {
		if input == expected {
			return nil
		}
		return errf(input)
	}
}

// YesConfirmation validates a y/N confirmation prompt.
func YesConfirmation(err error) func(string) error {
	return func(input string) error {
		if input == "y" {
			return nil
		}
		return err
	}
}

// IgnoreCanceled returns nil for user-canceled command errors.
func IgnoreCanceled(err error) error {
	if err == nil || errors.Is(err, errdef.ErrCanceled) {
		return nil
	}
	return err
}

// ReadConfirmation reads a lower-cased, trimmed confirmation line and respects cancellation.
func ReadConfirmation(ctx context.Context, in io.Reader) (string, error) {
	userIn := make(chan string, 1)
	go func() {
		reader := bufio.NewReader(in)
		input, _ := reader.ReadString('\n')
		userIn <- strings.ToLower(strings.TrimSpace(input))
	}()

	select {
	case <-ctx.Done():
		return "", errdef.ErrCanceled
	case input := <-userIn:
		return input, nil
	}
}
