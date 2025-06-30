package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tansive/tansive/internal/common/httpclient"
)

var (
	// Put command flags
	putCatalog   string
	putVariant   string
	putNamespace string
	putFile      string
	putData      string
)

// putCmd represents the put command
var putCmd = &cobra.Command{
	Use:   "put RESOURCE_PATH [flags]",
	Short: "Update a resource value by path",
	Long: `Update a resource value by path. The format is RESOURCE_PATH.
You can provide the data either through a file (-f) or directly (-d).

Examples:
  # Update a resource using a JSON file
  tansive put resources/path/to/resource -f data.json

  # Update a resource using inline JSON data
  tansive put resources/path/to/resource -d '{"name":"example","value":42}'

  # Update a resource in a specific context
  tansive put resources/path/to/resource -f data.json -c my-catalog -v my-variant -n my-namespace

  # Update a resource and output in JSON format
  tansive put resources/path/to/resource -f data.json -j`,
	Args: cobra.ExactArgs(1),
	RunE: putResourceValue,
}

// putResourceValue handles updating a resource value by path
// It validates the input and sends the update request to the server
func putResourceValue(cmd *cobra.Command, args []string) error {
	// Split the argument into resource type and name
	parts := strings.SplitN(args[0], "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid resource format. Expected <resourceType>/<resourceName>")
	}

	resourceType := parts[0]

	// Map the resource type to its URL format
	urlResourceType, err := MapResourceTypeToURL(resourceType)
	if err != nil {
		return err
	}

	if urlResourceType != "resources" {
		return fmt.Errorf("invalid resource type. Expected resources")
	}

	// Read input data
	var jsonData []byte
	if putFile != "" {
		// Clean and validate the file path
		cleanPath := filepath.Clean(putFile)

		jsonData, err = os.ReadFile(cleanPath)
		if err != nil {
			return fmt.Errorf("failed to read file: %v", err)
		}
	} else if putData != "" {
		jsonData = []byte(putData)
	} else {
		return fmt.Errorf("either --file or --data must be specified")
	}

	// Validate JSON
	var jsonObj map[string]interface{}
	if err := json.Unmarshal(jsonData, &jsonObj); err != nil {
		return fmt.Errorf("invalid JSON data: %v", err)
	}

	client := httpclient.NewClient(GetConfig())

	queryParams := make(map[string]string)
	if putCatalog != "" {
		queryParams["catalog"] = putCatalog
	}
	if putVariant != "" {
		queryParams["variant"] = putVariant
	}
	if putNamespace != "" {
		queryParams["namespace"] = putNamespace
	}

	resourcePath := "/" + urlResourceType + "/" + strings.TrimPrefix(parts[1], "/")
	response, err := client.UpdateResourceValue(resourcePath, jsonData, queryParams)
	if err != nil {
		return err
	}

	if jsonOutput {
		// Format as JSON with result and value
		output := map[string]any{
			"result": 1,
			"value":  response,
		}

		jsonBytes, err := json.MarshalIndent(output, "", "    ")
		if err != nil {
			return fmt.Errorf("failed to format JSON output: %v", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Println("Resource value updated successfully")
	}
	return nil
}

// init initializes the put command with its flags and adds it to the root command
func init() {
	rootCmd.AddCommand(putCmd)

	// Add flags
	putCmd.Flags().StringVarP(&putCatalog, "catalog", "c", "", "Catalog name")
	putCmd.Flags().StringVarP(&putVariant, "variant", "v", "", "Variant name")
	putCmd.Flags().StringVarP(&putNamespace, "namespace", "n", "", "Namespace name")
	putCmd.Flags().StringVarP(&putFile, "file", "f", "", "File containing JSON data")
	putCmd.Flags().StringVarP(&putData, "data", "d", "", "JSON data string")
	putCmd.MarkFlagsMutuallyExclusive("file", "data")
}
