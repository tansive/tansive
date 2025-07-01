package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tansive/tansive/internal/common/httpclient"
)

var (
	// Create command flags
	createCatalog   string
	createVariant   string
	createNamespace string
	ignoreErrors    bool
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create -f FILENAME [flags]",
	Short: "Create a resource from a file",
	Long: `Create a resource from a file. The resource type is determined by the 'kind' field in the YAML file.
Supported resource types include:
  - Catalogs
  - Variants
  - Namespaces
  - Views
  - Resources
  - Skillsets

Examples:
  # Create a new catalog
  tansive create -f catalog.yaml

  # Create a variant in a specific catalog
  tansive create -f variant.yaml -c my-catalog

  # Create a namespace in a catalog and variant
  tansive create -f namespace.yaml -c my-catalog -v my-variant

  # Create a resource in a specific context
  tansive create -f resource.yaml -c my-catalog -v my-variant -n my-namespace`,
	RunE: createResource,
}

// createResource handles the creation of a resource from a file
// It validates the input, loads the resource, and sends it to the server
func createResource(cmd *cobra.Command, args []string) error {
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
			printCreateStatus(statusValues)
		}
	}()

	return processResourcesByKind(resources, orderedResourceList, &statusValues)
}

// processResourcesByKind processes resources in the specified order by kind
func processResourcesByKind(resources map[string]ResourceList, orderedResourceList []string, statusValues *[]map[string]any) error {
	for _, kind := range orderedResourceList {
		resources, ok := resources[kind]
		if !ok {
			continue
		}
		if err := processResourcesOfKind(resources, statusValues); err != nil {
			return err
		}
	}
	return nil
}

// processResourcesOfKind processes all resources of a specific kind
func processResourcesOfKind(resources ResourceList, statusValues *[]map[string]any) error {
	for _, resource := range resources {
		kv, err := handleCreateResource(resource.Metadata, resource.JSON)
		if err != nil {
			*statusValues = append(*statusValues, createCreateErrorStatus(resource.Metadata, err))
			if !ignoreErrors {
				return ErrAlreadyHandled
			}
			continue
		}
		*statusValues = append(*statusValues, kv)
	}
	return nil
}

// createCreateErrorStatus creates an error status map for a failed resource creation operation
func createCreateErrorStatus(resource ResourceMetadata, err error) map[string]any {
	return map[string]any{
		"kind":    resource.Kind,
		"name":    resource.Metadata["name"],
		"created": false,
		"error":   err.Error(),
	}
}

// printCreateStatus prints the status of resource creation operations
func printCreateStatus(statusValues []map[string]any) {
	if jsonOutput {
		printJSON(statusValues)
		return
	}
	printHumanReadableCreateStatus(statusValues)
}

// printHumanReadableCreateStatus prints status in human-readable format
func printHumanReadableCreateStatus(statusValues []map[string]any) {
	for _, status := range statusValues {
		if isCreateSuccessStatus(status) {
			printCreateSuccessStatus(status)
		} else {
			printCreateErrorStatus(status)
		}
	}
}

// isCreateSuccessStatus checks if the status represents a created resource
func isCreateSuccessStatus(status map[string]any) bool {
	created, exists := status["created"]
	return exists && created.(bool)
}

// printCreateSuccessStatus prints a created resource status
func printCreateSuccessStatus(status map[string]any) {
	location, ok := status["location"].(string)
	if !ok {
		location = ""
	}
	okLabel.Fprintf(os.Stdout, "[OK] ")
	fmt.Fprintf(os.Stdout, "Created: %s\n", location)
}

// printCreateErrorStatus prints an error status for create operations
func printCreateErrorStatus(status map[string]any) {
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

func handleCreateResource(resource ResourceMetadata, jsonData []byte) (map[string]any, error) {
	resourceType, err := GetResourceType(resource.Kind)
	if err != nil {
		return nil, err
	}

	client := httpclient.NewClient(GetConfig())
	queryParams := make(map[string]string)
	if createCatalog != "" {
		queryParams["catalog"] = createCatalog
	}
	if createVariant != "" {
		queryParams["variant"] = createVariant
	}
	if createNamespace != "" {
		queryParams["namespace"] = createNamespace
	}

	_, location, err := client.CreateResource(resourceType, jsonData, queryParams)
	if err != nil {
		return nil, err
	}

	kv := map[string]any{
		"kind":     resource.Kind,
		"created":  true,
		"location": location,
		"name":     resource.Metadata["name"],
	}
	return kv, nil
}

// init initializes the create command with its flags and adds it to the root command
func init() {
	// Add flags to the create command
	createCmd.Flags().StringP("filename", "f", "", "Filename to use to create the resource")
	createCmd.MarkFlagRequired("filename")

	// Add context flags
	createCmd.Flags().StringVarP(&createCatalog, "catalog", "c", "", "Catalog name")
	createCmd.Flags().StringVarP(&createVariant, "variant", "v", "", "Variant name")
	createCmd.Flags().StringVarP(&createNamespace, "namespace", "n", "", "Namespace name")
	createCmd.Flags().BoolVarP(&ignoreErrors, "ignore-errors", "i", false, "Ignore errors and continue with the next resource")

	// Add the create command to the root command
	rootCmd.AddCommand(createCmd)
}
