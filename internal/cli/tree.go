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

	var catalogName string
	if getTreeCatalog != "" {
		catalogName = getTreeCatalog
	} else {
		catalogName = GetConfig().CurrentCatalog
	}

	if catalogName == "" {
		return fmt.Errorf("set a catalog first with `tansive set-catalog <catalog-name>`")
	}

	response, err := client.GetResource("catalogs", catalogName, queryParams, "")
	if err != nil {
		return err
	}

	var variantObjects []catalogmanager.VariantObject
	if err := json.Unmarshal(response, &variantObjects); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	root := &Node{Name: "📁 Catalog"}

	for _, v := range variantObjects {
		variant, err := decompressVariantObject(v)
		if err != nil {
			return err
		}

		variantNode := &Node{Name: "🧬 " + variant.Name}
		root.Children = append(root.Children, variantNode)

		// Build namespace set from variant.Namespaces
		knownNamespaces := map[string]struct{}{}
		for _, ns := range variant.Namespaces {
			knownNamespaces[ns] = struct{}{}
		}

		// Map to group by namespace
		nsMap := map[string]*Node{}

		process := func(objs []CatalogObject, category string, icon string) {
			// Build lookup set for namespaces in this variant
			nsSet := map[string]struct{}{}
			for _, n := range variant.Namespaces {
				nsSet[n] = struct{}{}
			}

			for _, obj := range objs {
				segments := strings.Split(obj.Path, "/")
				if len(segments) < 3 {
					continue // Not enough segments
				}

				if segments[1] != "--root--" {
					continue // Invalid prefix
				}

				var ns string
				var pathStart int

				// Determine if segments[2] is a namespace
				candidate := segments[2]
				if _, ok := nsSet[candidate]; ok {
					// It is a namespace
					ns = candidate
					pathStart = 3
				} else {
					// It is part of the path
					ns = "default"
					pathStart = 2
				}

				// Namespace node
				nsNode, ok := nsMap[ns]
				if !ok {
					nsNode = &Node{Name: "🌐 " + ns}
					nsMap[ns] = nsNode
					variantNode.Children = append(variantNode.Children, nsNode)
				}

				// Category node
				var catNode *Node
				for _, child := range nsNode.Children {
					if child.Name == icon+" "+category {
						catNode = child
						break
					}
				}
				if catNode == nil {
					catNode = &Node{Name: icon + " " + category}
					nsNode.Children = append(nsNode.Children, catNode)
				}

				insertPath(catNode, segments[pathStart:])
			}
		}

		process(variant.SkillSets, "SkillSets", "🧠")
		process(variant.Resources, "Resources", "📦")
	}

	for _, variantNode := range root.Children {
		for _, nsNode := range variantNode.Children {
			for _, catNode := range nsNode.Children {
				for _, child := range catNode.Children {
					collapseSingleChildFolders(child)
				}
			}
		}
	}

	fmt.Println(root.Name)
	for i, child := range root.Children {
		printTree(child, "", i == len(root.Children)-1)
	}

	return nil
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
