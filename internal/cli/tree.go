package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"encoding/base64"

	"github.com/golang/snappy"
	"github.com/spf13/cobra"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager"
	"github.com/tansive/tansive/internal/common/httpclient"
)

var (
	// Get-tree command flags
	getTreeCatalog string
)

type CatalogObject struct {
	Path string `json:"Path"`
}

// DecompressedVariantObject represents the structure after decompression
type DecompressedVariantObject struct {
	Name       string          `json:"name"`
	Namespaces []string        `json:"namespaces"`
	SkillSets  []CatalogObject `json:"skillsets"`
	Resources  []CatalogObject `json:"resources"`
}

// getTreeCmd represents the get-tree command
var getTreeCmd = &cobra.Command{
	Use:   "tree [flags]",
	Short: "Get a tree of objects in a catalog",
	Long: `Get a tree of objects in a catalog. This command retrieves all variants, skillsets, and resources 
organized in a tree structure for the catalog resolved from the bearer token.

Examples:
  # Get tree for the current catalog (resolved from bearer token)
  tansive tree`,
	Args: cobra.NoArgs,
	RunE: getTreeResource,
}

// getTreeResource handles the retrieval of a catalog tree
// It calls the catalog's Get method with tree=true query parameter
func getTreeResource(cmd *cobra.Command, args []string) error {
	client := httpclient.NewClient(GetConfig())

	queryParams := map[string]string{"tree": "true"}

	catalogName := getCatalogName()
	if catalogName == "" {
		return fmt.Errorf("set a catalog first with `tansive set-catalog <catalog-name>`")
	}

	response, err := client.GetResource("catalogs", catalogName, queryParams, "")
	if err != nil {
		return err
	}

	variantObjects, err := parseVariantObjects(response)
	if err != nil {
		return err
	}

	root := buildCatalogTree(variantObjects)

	// Collapse single child folders
	collapseTreeNodes(root)

	// Print the tree
	printCatalogTree(root)

	return nil
}

// getCatalogName returns the catalog name to use
func getCatalogName() string {
	if getTreeCatalog != "" {
		return getTreeCatalog
	}
	return GetConfig().CurrentCatalog
}

// parseVariantObjects parses the response into variant objects
func parseVariantObjects(response []byte) ([]catalogmanager.VariantObject, error) {
	var variantObjects []catalogmanager.VariantObject
	if err := json.Unmarshal(response, &variantObjects); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}
	return variantObjects, nil
}

// buildCatalogTree builds the complete catalog tree structure
func buildCatalogTree(variantObjects []catalogmanager.VariantObject) *Node {
	root := &Node{Name: "📁 Catalog"}

	for _, v := range variantObjects {
		variant, err := decompressVariantObject(v)
		if err != nil {
			// Log error but continue with other variants
			fmt.Printf("Warning: failed to decompress variant %s: %v\n", v.Name, err)
			continue
		}

		variantNode := buildVariantNode(variant)
		root.Children = append(root.Children, variantNode)
	}

	return root
}

// buildVariantNode builds a tree node for a single variant
func buildVariantNode(variant DecompressedVariantObject) *Node {
	variantNode := &Node{Name: "🧬 " + variant.Name}

	// Build namespace set for quick lookup
	nsSet := buildNamespaceSet(variant.Namespaces)

	// Map to group by namespace
	nsMap := map[string]*Node{}

	// Process skillsets and resources
	processObjects(variant.SkillSets, "SkillSets", "🧠", variantNode, nsMap, nsSet)
	processObjects(variant.Resources, "Resources", "📦", variantNode, nsMap, nsSet)

	return variantNode
}

// buildNamespaceSet creates a set of namespaces for quick lookup
func buildNamespaceSet(namespaces []string) map[string]struct{} {
	nsSet := map[string]struct{}{}
	for _, ns := range namespaces {
		nsSet[ns] = struct{}{}
	}
	return nsSet
}

// processObjects processes a list of catalog objects and builds the tree structure
func processObjects(objs []CatalogObject, category string, icon string, variantNode *Node, nsMap map[string]*Node, nsSet map[string]struct{}) {
	for _, obj := range objs {
		segments := strings.Split(obj.Path, "/")
		if !isValidPath(segments) {
			continue
		}

		ns, pathStart := determineNamespace(segments, nsSet)

		// Get or create namespace node
		nsNode := getOrCreateNamespaceNode(ns, variantNode, nsMap)

		// Get or create category node
		catNode := getOrCreateCategoryNode(nsNode, category, icon)

		// Insert the path
		insertPath(catNode, segments[pathStart:])
	}
}

// isValidPath checks if a path has the minimum required segments and valid prefix
func isValidPath(segments []string) bool {
	if len(segments) < 3 {
		return false // Not enough segments
	}
	return segments[1] == "--root--" // Valid prefix
}

// determineNamespace determines the namespace and path start index
func determineNamespace(segments []string, nsSet map[string]struct{}) (string, int) {
	candidate := segments[2]
	if _, ok := nsSet[candidate]; ok {
		// It is a namespace
		return candidate, 3
	}
	// It is part of the path
	return "default", 2
}

// getOrCreateNamespaceNode gets or creates a namespace node
func getOrCreateNamespaceNode(ns string, variantNode *Node, nsMap map[string]*Node) *Node {
	nsNode, ok := nsMap[ns]
	if !ok {
		nsNode = &Node{Name: "🌐 " + ns}
		nsMap[ns] = nsNode
		variantNode.Children = append(variantNode.Children, nsNode)
	}
	return nsNode
}

// getOrCreateCategoryNode gets or creates a category node
func getOrCreateCategoryNode(nsNode *Node, category string, icon string) *Node {
	for _, child := range nsNode.Children {
		if child.Name == icon+" "+category {
			return child
		}
	}
	catNode := &Node{Name: icon + " " + category}
	nsNode.Children = append(nsNode.Children, catNode)
	return catNode
}

// collapseTreeNodes collapses single child folders in the entire tree
func collapseTreeNodes(root *Node) {
	for _, variantNode := range root.Children {
		for _, nsNode := range variantNode.Children {
			for _, catNode := range nsNode.Children {
				for _, child := range catNode.Children {
					collapseSingleChildFolders(child)
				}
			}
		}
	}
}

// printCatalogTree prints the catalog tree
func printCatalogTree(root *Node) {
	fmt.Println(root.Name)
	for i, child := range root.Children {
		printTree(child, "", i == len(root.Children)-1)
	}
}

func decompressVariantObject(obj catalogmanager.VariantObject) (DecompressedVariantObject, error) {
	decompressedObj := DecompressedVariantObject{
		Name:       obj.Name,
		Namespaces: obj.Namespaces,
	}

	// Decompress and parse skillsets
	if obj.SkillSets != "" {
		base64Decoded, err := base64.StdEncoding.DecodeString(obj.SkillSets)
		if err != nil {
			return DecompressedVariantObject{}, fmt.Errorf("failed to base64 decode skillsets for variant %s: %v", obj.Name, err)
		}
		decompressedSkillsets, err := snappy.Decode(nil, base64Decoded)
		if err != nil {
			return DecompressedVariantObject{}, fmt.Errorf("failed to snappy decompress skillsets for variant %s: %v", obj.Name, err)
		}
		var skillsets []CatalogObject
		if err := json.Unmarshal(decompressedSkillsets, &skillsets); err != nil {
			return DecompressedVariantObject{}, fmt.Errorf("failed to parse decompressed skillsets for variant %s: %v", obj.Name, err)
		}
		decompressedObj.SkillSets = skillsets
	}

	// Decompress and parse resources
	if obj.Resources != "" {
		base64Decoded, err := base64.StdEncoding.DecodeString(obj.Resources)
		if err != nil {
			return DecompressedVariantObject{}, fmt.Errorf("failed to base64 decode resources for variant %s: %v", obj.Name, err)
		}
		decompressedResources, err := snappy.Decode(nil, base64Decoded)
		if err != nil {
			return DecompressedVariantObject{}, fmt.Errorf("failed to snappy decompress resources for variant %s: %v", obj.Name, err)
		}
		var resources []CatalogObject
		if err := json.Unmarshal(decompressedResources, &resources); err != nil {
			return DecompressedVariantObject{}, fmt.Errorf("failed to parse decompressed resources for variant %s: %v", obj.Name, err)
		}
		decompressedObj.Resources = resources
	}

	return decompressedObj, nil
}

// Insert segments into tree
func insertPath(root *Node, segments []string) {
	if len(segments) == 0 {
		return
	}
	// Find or create child
	for _, child := range root.Children {
		if child.Name == segments[0] {
			insertPath(child, segments[1:])
			return
		}
	}
	// Not found - create new
	newChild := &Node{Name: segments[0]}
	root.Children = append(root.Children, newChild)
	insertPath(newChild, segments[1:])
}

type Node struct {
	Name     string
	Children []*Node
}

func printTree(node *Node, prefix string, isLast bool) {
	// Determine which branch prefix to use
	var branch string
	if isLast {
		branch = "└── "
	} else {
		branch = "├── "
	}

	// Print current node
	fmt.Printf("%s%s%s\n", prefix, branch, node.Name)

	// Prepare prefix for children
	var newPrefix string
	if isLast {
		newPrefix = prefix + "    "
	} else {
		newPrefix = prefix + "│   "
	}

	for i, child := range node.Children {
		printTree(child, newPrefix, i == len(node.Children)-1)
	}
}

func collapseSingleChildFolders(node *Node) {
	for _, child := range node.Children {
		collapseSingleChildFolders(child)
	}

	// Keep collapsing while there's exactly one child and it's not a leaf
	for len(node.Children) == 1 && len(node.Children[0].Children) > 0 {
		child := node.Children[0]
		// Merge names with "/"
		node.Name = node.Name + "/" + child.Name
		// Adopt the grand-children
		node.Children = child.Children
	}
}

// init initializes the get-tree command with its flags and adds it to the root command
func init() {
	rootCmd.AddCommand(getTreeCmd)

	// Add flags
	getTreeCmd.Flags().StringVarP(&getTreeCatalog, "catalog", "c", "", "Catalog name (alternative to positional argument)")
}
