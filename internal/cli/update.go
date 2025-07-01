package cli

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/tansive/tansive/internal/common/httpclient"
)

var (
	// Update command flags
	updateCatalog   string
	updateVariant   string
	updateNamespace string
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "apply -f FILENAME [flags]",
	Short: "Apply a resource from a file (create if not exists, update if exists)",
	Long: `Apply a resource from a file. The resource type is determined by the 'kind' field in the YAML file.
This command follows the Kubernetes-style apply pattern - it will create the resource if it doesn't exist,
or update it if it already exists.

Supported resource types include:
  - Catalogs
  - Variants
  - Namespaces
  - Views
  - Resources
  - Skillsets

Examples:
  # Apply a catalog configuration
  tansive apply -f catalog.yaml

  # Apply a variant in a specific catalog
  tansive apply -f variant.yaml -c my-catalog

  # Apply a namespace in a catalog and variant
  tansive apply -f namespace.yaml -c my-catalog -v my-variant

  # Apply a resource in a specific context
  tansive apply -f resource.yaml -c my-catalog -v my-variant -n my-namespace

  # Apply a resource and output in JSON format
  tansive apply -f resource.yaml -j`,
	RunE: updateResource,
}

// updateResource handles applying a resource from a file
// It attempts to create the resource first, then updates if it already exists
func updateResource(cmd *cobra.Command, args []string) error {
	filename, err := cmd.Flags().GetString("filename")
	if err != nil {
		return err
	}
	if filename == "" {
		return fmt.Errorf("filename is required")
	}

	resources, err := LoadResourceFromMultiYAMLFile(filename)
	if err != nil {
		return err
	}

	orderedResourceList := []string{
		KindCatalog,
		KindVariant,
		KindNamespace,
		KindView,
		KindSkillset,
		KindResource,
	}

	var statusValues []map[string]any
	defer func() {
		if len(statusValues) > 0 {
			printUpdateStatus(statusValues)
		}
	}()

	return processResourcesInOrder(resources, orderedResourceList, &statusValues)
}

// processResourcesInOrder processes resources in the specified order by kind
func processResourcesInOrder(resources map[string]ResourceList, orderedResourceList []string, statusValues *[]map[string]any) error {
	for _, kind := range orderedResourceList {
		resources, ok := resources[kind]
		if !ok {
			continue
		}
		if err := processResourcesOfSingleKind(resources, statusValues); err != nil {
			return err
		}
	}
	return nil
}

// processResourcesOfSingleKind processes all resources of a specific kind
func processResourcesOfSingleKind(resources ResourceList, statusValues *[]map[string]any) error {
	for _, resource := range resources {
		kv, err := handleUpdateResource(resource.Metadata, resource.JSON)
		if err != nil {
			*statusValues = append(*statusValues, createErrorStatus(resource.Metadata, err))
			if !ignoreErrors {
				return ErrAlreadyHandled
			}
			continue
		}
		*statusValues = append(*statusValues, kv)
	}
	return nil
}

// createErrorStatus creates an error status map for a failed resource operation
func createErrorStatus(resource ResourceMetadata, err error) map[string]any {
	return map[string]any{
		"kind":    resource.Kind,
		"name":    resource.Metadata["name"],
		"updated": false,
		"error":   err.Error(),
	}
}

// printUpdateStatus prints the status of resource operations
func printUpdateStatus(statusValues []map[string]any) {
	if jsonOutput {
		printJSON(statusValues)
		return
	}
	printHumanReadableStatus(statusValues)
}

// printHumanReadableStatus prints status in human-readable format
func printHumanReadableStatus(statusValues []map[string]any) {
	for _, status := range statusValues {
		if isCreatedStatus(status) {
			printCreatedStatus(status)
		} else if isUpdatedStatus(status) {
			printUpdatedStatus(status)
		} else {
			printErrorStatus(status)
		}
	}
}

// isCreatedStatus checks if the status represents a created resource
func isCreatedStatus(status map[string]any) bool {
	created, exists := status["created"]
	return exists && created.(bool)
}

// isUpdatedStatus checks if the status represents an updated resource
func isUpdatedStatus(status map[string]any) bool {
	updated, exists := status["updated"]
	return exists && updated.(bool)
}

// printCreatedStatus prints a created resource status
func printCreatedStatus(status map[string]any) {
	location, ok := status["location"].(string)
	if !ok {
		location = ""
	}
	okLabel.Fprintf(os.Stdout, "[OK] ")
	fmt.Fprintf(os.Stdout, "Created: %s\n", location)
}

// printUpdatedStatus prints an updated resource status
func printUpdatedStatus(status map[string]any) {
	name, exists := status["name"]
	if !exists {
		name = ""
	}
	okLabel.Fprintf(os.Stdout, "[OK] ")
	fmt.Fprintf(os.Stdout, "Updated: %s: %s\n", status["kind"], name)
}

// printErrorStatus prints an error status
func printErrorStatus(status map[string]any) {
	name, exists := status["name"]
	if !exists {
		name = ""
	}
	if !ignoreErrors {
		errorLabel.Fprintf(os.Stderr, "[ERROR] ")
		fmt.Fprintf(os.Stderr, "%s: %s: %s\n", status["kind"], name, status["error"])
	} else {
		errorLabel.Fprintf(os.Stdout, "[ERROR] ")
		fmt.Fprintf(os.Stdout, "%s: %s: %s\n", status["kind"], name, status["error"])
	}
}

func handleUpdateResource(resource ResourceMetadata, jsonData []byte) (map[string]any, error) {
	resourceType, err := GetResourceType(resource.Kind)
	if err != nil {
		return nil, err
	}
	client := httpclient.NewClient(GetConfig())
	queryParams := make(map[string]string)
	if updateCatalog != "" {
		queryParams["catalog"] = updateCatalog
	}
	if updateVariant != "" {
		queryParams["variant"] = updateVariant
	}
	if updateNamespace != "" {
		queryParams["namespace"] = updateNamespace
	}

	// First try to create the resource
	_, location, err := client.CreateResource(resourceType, jsonData, queryParams)
	if err != nil {
		// If we get a conflict, try to update instead
		if httpErr, ok := err.(*httpclient.HTTPError); ok && httpErr.StatusCode == http.StatusConflict {
			objectType := ""
			if resourceType == "resources" {
				objectType = "definition"
			}
			_, err = client.UpdateResource(resourceType, jsonData, queryParams, objectType)
			if err != nil {
				return nil, fmt.Errorf("failed to update resource: %v", err)
			}
			kv := map[string]any{
				"kind":    resource.Kind,
				"updated": true,
				"name":    resource.Metadata["name"],
			}
			return kv, nil
		}
		return nil, fmt.Errorf("failed to create resource: %v", err)
	}

	kv := map[string]any{
		"kind":     resource.Kind,
		"created":  true,
		"location": location,
		"name":     resource.Metadata["name"],
	}
	return kv, nil
}

// init initializes the update command with its flags and adds it to the root command
func init() {
	// Add flags to the update command
	updateCmd.Flags().StringP("filename", "f", "", "Filename to use to update the resource")
	updateCmd.MarkFlagRequired("filename")

	// Add context flags
	updateCmd.Flags().StringVarP(&updateCatalog, "catalog", "c", "", "Catalog name")
	updateCmd.Flags().StringVarP(&updateVariant, "variant", "v", "", "Variant name")
	updateCmd.Flags().StringVarP(&updateNamespace, "namespace", "n", "", "Namespace name")

	// Add the update command to the root command
	rootCmd.AddCommand(updateCmd)
}
