package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/elee1766/poundai/pkg/config"
	"github.com/elee1766/poundai/pkg/provider"
)

func runDoctor(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stdout)
	configPath := fs.String("config", "", "path to config file")
	service := fs.String("service", "", "service profile to test")
	offline := fs.Bool("offline", false, "validate configuration without contacting the provider")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("doctor: unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	path := resolveConfigPath(*configPath)
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "ok config %s\n", path)

	name := *service
	if name == "" {
		name = os.Getenv("POUNDAI_SERVICE")
	}
	if name == "" {
		name = cfg.Service
	}
	svc, err := cfg.Active(name)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "ok service %s (%s/%s)\n", name, svc.Provider, svc.Model)

	prov, err := provider.New(svc)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, "ok provider configuration")
	if *offline {
		return nil
	}

	timeout := svc.Timeout.Std(provider.DefaultTimeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result, err := prov.Complete(ctx, []provider.Message{
		{Role: "system", Content: "Reply with only the word ok."},
		{Role: "user", Content: "Connectivity check"},
	})
	if err != nil {
		return fmt.Errorf("provider connectivity: %w", err)
	}
	if strings.TrimSpace(result) == "" {
		return fmt.Errorf("provider connectivity: empty response")
	}
	fmt.Fprintln(stdout, "ok provider connectivity")
	return nil
}
