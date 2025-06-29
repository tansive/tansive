package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tansive/tansive/internal/common/httpclient"
	"sigs.k8s.io/yaml"
)

var (
	// Get command flags
	getCatalog   string
	getVariant   string
	getNamespace string
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get RESOURCE_PATH [flags]",
	Short: "Get a resource value by path",
	Long: `Get a resource value by path. The format is RESOURCE_PATH.
This command only works with resources and returns their current values.

Examples:
  # Get a resource value
  tansive get resources/path/to/resource

  # Get a resource value in a specific catalog
  tansive get resources/path/to/resource -c my-catalog

  # Get a resource value in a specific context
  tansive get resources/path/to/resource -c my-catalog -v my-variant -n my-namespace

  # Get a resource value in JSON format
  tansive get resources/path/to/resource -j`,
	Args: cobra.ExactArgs(1),
	RunE: getResourceValue,
}

// getResourceValue handles retrieving a resource value by path
// It validates the input and formats the output in YAML or JSON
func getResourceValue(cmd *cobra.Command, args []string) error {
	// Split the argument into resource type and name
	parts := strings.SplitN(args[0], "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid resource format. Expected <resourceType>/<resourceName>")
	}

	resourceType := parts[0]
	resourcePath := parts[1]

	// Map the resource type to its URL format
	urlResourceType, err := MapResourceTypeToURL(resourceType)
	if err != nil {
		return err
	}

	client := httpclient.NewClient(GetConfig())

	queryParams := make(map[string]string)
	if getCatalog != "" {
		queryParams["catalog"] = getCatalog
	}
	if getVariant != "" {
		queryParams["variant"] = getVariant
	}
	if getNamespace != "" {
		queryParams["namespace"] = getNamespace
	}

	if urlResourceType != "resources" {
		return fmt.Errorf("invalid resource type. Expected resources")
	}

	response, err := client.GetResource(urlResourceType, resourcePath, queryParams, "")
	if err != nil {
		return err
	}

	var responseData map[string]any
	if err := json.Unmarshal(response, &responseData); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	if jsonOutput {
		// Format as JSON with result and value
		output := map[string]any{
			"result": 1,
			"value":  responseData,
		}

		jsonBytes, err := json.MarshalIndent(output, "", "    ")
		if err != nil {
			return fmt.Errorf("failed to format JSON output: %v", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		// Convert to YAML
		yamlBytes, err := yaml.Marshal(responseData)
		if err != nil {
			return fmt.Errorf("failed to convert to YAML: %v", err)
		}
		fmt.Println(string(yamlBytes))
	}
	return nil
}

// init initializes the get command with its flags and adds it to the root command
func init() {
	rootCmd.AddCommand(getCmd)

	// Add flags
	getCmd.Flags().StringVarP(&getCatalog, "catalog", "c", "", "Catalog name")
	getCmd.Flags().StringVarP(&getVariant, "variant", "v", "", "Variant name")
	getCmd.Flags().StringVarP(&getNamespace, "namespace", "n", "", "Namespace name")
}
