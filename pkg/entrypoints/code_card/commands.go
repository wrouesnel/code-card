package code_card

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	"github.com/wrouesnel/code-card/pkg/codecard"
	"github.com/wrouesnel/ctxstdio"
	logutil "github.com/wrouesnel/go.logutil"
	"go.uber.org/zap"
)

// DefaultSalt is the default salt used for encoded cards, matching the original implementation.
const DefaultSalt = "1337code8Ackz0rs"

// RandomCmd generates a new random code card.
type RandomCmd struct{}

// Run generates the card.
func (r *RandomCmd) Run(ctx context.Context, cli *CLIConfig) error {
	table := codecard.RandomTable()
	return writeCard(ctx, cli, &table)
}

// EncodedCmd encodes an existing PIN to a keyword on a code card.
type EncodedCmd struct {
	Salt    string `default:"${default_salt}"           env:"${env_prefix}_ENCODED_SALT"                                     help:"Salt value for code cards"`
	Keyword string `env:"${env_prefix}_ENCODED_KEYWORD" help:"Keyword which will produce the code (prompted for if not set)"`
	Code    string `env:"${env_prefix}_ENCODED_CODE"    help:"Numeric PIN which will be encoded (prompted for if not set)"`
}

// Run generates the card.
func (e *EncodedCmd) Run(ctx context.Context, cli *CLIConfig) error {
	prompt := newPrompter(ctx)
	var err error
	if e.Keyword == "" {
		if e.Keyword, err = prompt.Secret("Keyword"); err != nil {
			return errors.Wrap(err, "reading keyword")
		}
	}
	if e.Code == "" {
		if e.Code, err = prompt.Secret("Code"); err != nil {
			return errors.Wrap(err, "reading code")
		}
	}

	table, err := codecard.SemiRandomTable(e.Salt, e.Code, e.Keyword)
	if err != nil {
		return err
	}
	return writeCard(ctx, cli, &table)
}

// writeCard renders table and writes it to the configured output. The card is rendered fully
// before the output is opened so failures don't leave partial files.
func writeCard(ctx context.Context, cli *CLIConfig, table *codecard.Table) error {
	l := logutil.FromCtx(ctx)

	var buf bytes.Buffer
	if err := codecard.Render(&buf, table, codecard.RenderOptions{Title: cli.Title}); err != nil {
		return errors.Wrap(err, "rendering card")
	}

	if cli.OutputFile.String() == "-" {
		_, err := ctxstdio.StdOut(ctx).Write(buf.Bytes())
		return errors.Wrap(err, "writing card to stdout")
	}

	if err := cli.OutputFile.WriteFile(buf.Bytes()); err != nil {
		return errors.Wrap(err, "writing card")
	}
	l.Debug("Wrote code card", zap.String("filename", cli.OutputFile.String()))
	return nil
}
