package code_card

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/wrouesnel/code-card/version"
	"github.com/wrouesnel/ctxstdio"
	"github.com/wrouesnel/kongutil"

	"github.com/chigopher/pathlib"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/alecthomas/kong"
)

// Name is the name of this subapplication.
const Name = "code-card"

// CLIConfig is the root command line configuration parsed by kong.
type CLIConfig struct {
	Version kong.VersionFlag `help:"Show version number"`

	Title      string        `help:"Title for the card"`
	OutputFile *pathlib.Path `default:"code-card.pdf"   help:"Output filename ('-' for stdout)"`

	Random  RandomCmd  `cmd:"" help:"Generate a new random code card"`
	Encoded EncodedCmd `cmd:"" help:"Encode an existing PIN to a keyword on a code card"`
}

// kongExit carries an exit code requested by kong (e.g. for --help or --version) out of the
// parser so Entrypoint can return it instead of the process exiting.
type kongExit struct {
	code int
}

// Entrypoint is the real application entrypoint. This structure allows test packages to E2E-style tests invoking commmands
// as though they are on the command line, but using built-in coverage tools. Stub-main under the `cmd` package calls this
// function with os.Args[1:].
func Entrypoint(ctx context.Context, args []string) (exitCode int) {
	stdOut := ctxstdio.StdOut(ctx)
	stdErr := ctxstdio.StdErr(ctx)

	defer func() {
		if r := recover(); r != nil {
			exit, ok := r.(kongExit)
			if !ok {
				panic(r)
			}
			exitCode = exit.code
		}
	}()

	appCtx, appCancel := context.WithCancel(ctx)
	defer appCancel()

	// Command line parsing can now happen
	envPrefix := strings.ReplaceAll(strings.ToUpper(version.Name), "-", "_")
	vars := kong.Vars{
		"version":      version.Version,
		"env_prefix":   envPrefix,
		"default_salt": DefaultSalt,
	}

	// cli is the root entrypoint config parsed by kong and made available to subcommands via
	// a binding.
	var cli CLIConfig

	parser, err := kong.New(&cli,
		kong.Name(Name),
		kong.Description(version.Description),
		kong.DefaultEnvars(envPrefix),
		kong.Writers(stdOut, stdErr),
		kong.Exit(func(code int) { panic(kongExit{code: code}) }),
		kongutil.PathlibMapper(),
		vars)
	if err != nil {
		_, _ = fmt.Fprintf(stdErr, "Failure while building command line parser: %v\n", err)
		return 1
	}

	kongCtx, err := parser.Parse(args)
	parser.FatalIfErrorf(err)

	// Only warnings and errors are logged, so nothing is printed on success.
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	logger := zap.New(zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(stdErr), zap.WarnLevel))
	defer func() { _ = logger.Sync() }()

	// Install as the global logger
	defer zap.ReplaceGlobals(logger)()

	logger.Debug("Configuring signal handling")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	// Stop signal delivery on return so repeated calls (e.g. from tests) don't leak handlers.
	defer func() {
		signal.Stop(sigCh)
		close(sigCh)
	}()
	sigCtx, cancelFn := context.WithCancel(appCtx)
	defer cancelFn()
	go func() {
		for sig := range sigCh {
			logger.Info("Caught signal - exiting", zap.String("signal", sig.String()))
			cancelFn()
		}
		logger.Debug("Signal handling channel closed")
	}()

	// Main application context
	kongCtx.BindTo(sigCtx, (*context.Context)(nil))
	// Main command line
	kongCtx.Bind(&cli)
	// kongCtx if needed.
	kongCtx.Bind(kongCtx)

	if err := kongCtx.Run(); err != nil {
		// Report command errors the same way as command line errors.
		parser.Errorf("%s", err)
		return 1
	}
	return 0
}
