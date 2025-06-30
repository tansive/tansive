package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	jsonOutput bool
	configFile string
)

var ErrAlreadyHandled = errors.New("already handled")

var okLabel = color.New(color.FgGreen)
var errorLabel = color.New(color.FgRed)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tansive [command] [flags]",
	Short: "Tansive CLI - A command line interface for managing Tansive resources",
	Long: `Tansive CLI is a command line interface for managing Tansive resources.
It allows you to create, read, update, and delete resources using YAML files.
The CLI supports various resource types including catalogs, variants, namespaces, and workspaces.

Examples:
  # Create a new catalog
  tansive create -f catalog.yaml

  # Get a resource value
  tansive get resources/path/to/resource

  # Delete a catalog
  tansive delete catalog/my-catalog

  # List all resources
  tansive list resources`,
	PersistentPreRun: preRunHandlePersistents,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Set up persistent flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "", "", "Path to configuration file to override default")
	rootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON format")

	// Add commands
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newLoginCmd())
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.SilenceErrors = true // Prevent Cobra from printing the error
	rootCmd.SilenceUsage = true  // Prevent Cobra from printing usage on error

	err := rootCmd.Execute()
	if err != nil {
		if errors.Is(err, ErrAlreadyHandled) {
			os.Exit(1)
		}
		if jsonOutput {
			kv := map[string]string{
				"error": err.Error(),
			}
			printJSON(kv)
		} else {
			errorLabel.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}

// preRunHandlePersistents handles persistent flags and configuration loading before command execution
func preRunHandlePersistents(cmd *cobra.Command, args []string) {
	// if a config file is provided, load config from config file
	if configFile == "" {
		var err error
		configFile, err = GetDefaultConfigPath()
		if err != nil {
			errorLabel.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	isConfig := false
	c := cmd
	for c != nil {
		if c.Name() == "config" || c.Name() == "version" || c.Name() == "status" {
			isConfig = true
			break
		}
		c = c.Parent()
	}

	if !isConfig {
		if err := LoadConfig(configFile); err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Tansive config file not found. Configure tansive with \"tansive config create\" first.")
				os.Exit(1)
			} else {
				fmt.Printf("%s\n", err.Error())
				os.Exit(1)
			}
		}
	}
}

// newVersionCmd creates and returns a new version command
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of tansive-cli",
		Run: func(cmd *cobra.Command, args []string) {
			// Get the config file path
			configPath, err := GetDefaultConfigPath()
			if err != nil {
				configPath = "unknown"
			}

			if jsonOutput {
				kv := map[string]string{
					"version":     getCLIVersion(),
					"config_file": configPath,
				}
				printJSON(kv)
			} else {
				cmd.Printf("tansive CLI %s\n", getCLIVersion())
				cmd.Printf("Config file: %s\n", configPath)
			}
		},
	}
}

// printJSON prints the given map as JSON to stdout
func printJSON(data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonData))
}

// getCLIVersion returns the current CLI version
func getCLIVersion() string {
	return "v0.1.0-alpha.1"
}
