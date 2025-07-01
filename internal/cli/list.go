package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tansive/tansive/internal/common/httpclient"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	// List command flags
	listCatalog   string
	listVariant   string
	listNamespace string
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list RESOURCE_TYPE [flags]",
	Short: "List resources of a specific type",
	Long: `List resources of a specific type. Supported resource types include:
  - catalogs
  - variants
  - namespaces
  - views
  - resources
  - skillsets
  - sessions

Examples:
  # List all catalogs
  tansive list catalogs

  # List variants in a catalog
  tansive list variants -c my-catalog

  # List namespaces in a catalog and variant
  tansive list namespaces -c my-catalog -v my-variant

  # List views in a specific context
  tansive list views -c my-catalog -v my-variant

  # List resources in a specific context
  tansive list resources -c my-catalog -v my-variant -n my-namespace

  # List skillsets in a specific context
  tansive list skillsets -c my-catalog -v my-variant

  # List resources in JSON format
  tansive list resources -j

  # List catalogs in JSON format
  tansive list catalogs -j`,
	Args: cobra.ExactArgs(1),
	RunE: listResources,
}

// listResources handles listing resources of a specific type
// It retrieves the resources and formats the output based on the resource type
func listResources(cmd *cobra.Command, args []string) error {
	resourceType := args[0]

	// Map the resource type to its URL format
	urlResourceType, err := MapResourceTypeToURL(resourceType)
	if err != nil {
		return err
	}

	client := httpclient.NewClient(GetConfig())

	queryParams := make(map[string]string)
	if listCatalog != "" {
		queryParams["catalog"] = listCatalog
	}
	if listVariant != "" {
		queryParams["variant"] = listVariant
	}
	if listNamespace != "" {
		queryParams["namespace"] = listNamespace
	}

	response, err := client.ListResources(urlResourceType, queryParams)
	if err != nil {
		return err
	}

	// Use unified printing function for all resource types
	return printResourceList(urlResourceType, response)
}

// init initializes the list command with its flags and adds it to the root command
func init() {
	rootCmd.AddCommand(listCmd)

	// Add flags
	listCmd.Flags().StringVarP(&listCatalog, "catalog", "c", "", "Catalog name")
	listCmd.Flags().StringVarP(&listVariant, "variant", "v", "", "Variant name")
	listCmd.Flags().StringVarP(&listNamespace, "namespace", "n", "", "Namespace name")
}

// printResourceList formats and prints resources in either JSON or human-readable format
// This unified function handles different response formats for various resource types
func printResourceList(resourceType string, response []byte) error {
	if jsonOutput {
		return printResourceListJSON(resourceType, response)
	}
	return printResourceListHumanReadable(resourceType, response)
}

// printResourceListJSON handles JSON output formatting for different resource types
func printResourceListJSON(resourceType string, response []byte) error {
	if resourceType == "catalogs" {
		return printCatalogsJSON(response)
	}
	return printOtherResourcesJSON(response)
}

// printCatalogsJSON formats catalog names as JSON output
func printCatalogsJSON(response []byte) error {
	var catalogNames []string
	if err := json.Unmarshal(response, &catalogNames); err != nil {
		return fmt.Errorf("failed to parse catalog names: %v", err)
	}

	output := map[string]any{
		"result": 1,
		"value":  catalogNames,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to format JSON output: %v", err)
	}
	fmt.Println(string(jsonBytes))
	return nil
}

// printOtherResourcesJSON formats other resource types as JSON output
func printOtherResourcesJSON(response []byte) error {
	var responseData map[string]any
	if err := json.Unmarshal(response, &responseData); err != nil {
		return fmt.Errorf("failed to parse response")
	}

	output := map[string]any{
		"result": 1,
		"value":  responseData,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to format JSON output: %v", err)
	}
	fmt.Println(string(jsonBytes))
	return nil
}

// printResourceListHumanReadable handles human-readable output formatting for different resource types
func printResourceListHumanReadable(resourceType string, response []byte) error {
	fmt.Printf("%s:\n", cases.Title(language.English).String(resourceType))

	switch resourceType {
	case "catalogs":
		return printCatalogsHumanReadable(response)
	case "views":
		return printViewsHumanReadable(response)
	default:
		return printOtherResourcesHumanReadable(resourceType, response)
	}
}

// printCatalogsHumanReadable formats catalog names in human-readable format
func printCatalogsHumanReadable(response []byte) error {
	var catalogNames []string
	if err := json.Unmarshal(response, &catalogNames); err != nil {
		return fmt.Errorf("failed to parse catalog names: %v", err)
	}
	for _, name := range catalogNames {
		fmt.Printf("- %s\n", name)
	}
	return nil
}

// printViewsHumanReadable formats views in human-readable format
func printViewsHumanReadable(response []byte) error {
	var responseData map[string]any
	if err := json.Unmarshal(response, &responseData); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}
	if views, ok := responseData["views"].([]any); ok {
		for _, item := range views {
			if viewMap, ok := item.(map[string]any); ok {
				if name, ok := viewMap["name"].(string); ok {
					fmt.Printf("- %s\n", name)
				}
			}
		}
	}
	return nil
}

// printOtherResourcesHumanReadable formats other resource types in human-readable format
func printOtherResourcesHumanReadable(resourceType string, response []byte) error {
	var responseData map[string]any
	if err := json.Unmarshal(response, &responseData); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}
	if items, ok := responseData[resourceType].([]any); ok {
		for _, item := range items {
			if itemMap, ok := item.(map[string]any); ok {
				if name, ok := itemMap["name"].(string); ok {
					fmt.Printf("- %s\n", name)
				}
			}
		}
	} else {
		// If no structured format found, print the raw response
		fmt.Printf("Raw response: %s\n", string(response))
	}
	return nil
}
