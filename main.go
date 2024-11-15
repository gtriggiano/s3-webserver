package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/gtriggiano/s3-webserver/pkg/cli"
	"github.com/gtriggiano/s3-webserver/pkg/utils"
	"github.com/gtriggiano/s3-webserver/pkg/version"
	"github.com/joho/godotenv"

	"github.com/spf13/cobra"
)

func main() {
	godotenv.Load()

	root := utils.DefaultCommand(&cobra.Command{
		Use:              version.Progname,
		Short:            "Webserver exposing static files from S3",
		Version:          fmt.Sprintf("%s/%s, built %s", version.Version, version.Sha, version.BuildDate),
		TraverseChildren: true,
	})

	root.AddCommand(utils.DefaultCommand(cli.StartCommand()))

	if err := root.Execute(); err != nil {
		if msg := err.Error(); msg != "" {
			fmt.Fprintf(os.Stderr, "%s: %s\n", version.Progname, msg)
		}

		var exit *utils.ExitError
		if errors.As(err, &exit) {
			os.Exit(int(exit.Code))
		}

		os.Exit(int(utils.EX_FAIL))
	}
}
