package utils

import "github.com/spf13/cobra"

// DefaultCommand applies default settings to a cobra.Command to improve the help output.
func DefaultCommand(c *cobra.Command) *cobra.Command {
	c.SilenceUsage = true
	c.SilenceErrors = true
	c.DisableFlagsInUseLine = true

	return c
}
