package codegen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// StructInfo contains information about a struct with encx tags
type StructInfo struct {
	PackageName       string
	StructName        string
	SourceFile        string
	Fields            []FieldInfo
	HasEncxTags       bool
	GenerationOptions map[string]string // From //encx:options comments
	RequiredImports   map[string]string // package name -> import path
}

// FieldInfo contains information about a field with encx tags
type FieldInfo struct {
	Name             string
	Type             string
	EncxTags         []string
	IsValid          bool
	ValidationErrors []string
}

// DiscoveryConfig holds configuration for struct discovery
type DiscoveryConfig struct {
	SkipPackages []string
}

// DiscoverStructs discovers structs with encx tags in the given package path
func DiscoverStructs(packagePath string, config *DiscoveryConfig) ([]StructInfo, error) {
	var structs []StructInfo

	// Parse all .go files in the package
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, packagePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for pkgName, pkg := range pkgs {
		// Skip test packages
		if strings.HasSuffix(pkgName, "_test") {
			continue
		}

		// First pass: collect all struct definitions across all files in the package
		structDefs := make(map[string]*ast.StructType)
		fileImportsMap := make(map[string]map[string]string)

		for fileName, file := range pkg.Files {
			fileImportsMap[fileName] = extractImports(file)

			ast.Inspect(file, func(n ast.Node) bool {
				if genDecl, ok := n.(*ast.GenDecl); ok {
					for _, spec := range genDecl.Specs {
						if typeSpec, ok := spec.(*ast.TypeSpec); ok {
							if structType, ok := typeSpec.Type.(*ast.StructType); ok {
								structDefs[typeSpec.Name.Name] = structType
							}
						}
					}
				}
				return true
			})
		}

		// Second pass: analyze structs with embedded field resolution
		for fileName, file := range pkg.Files {
			fileImports := fileImportsMap[fileName]
			structs = append(structs, discoverStructsInFile(fset, fileName, file, pkgName, fileImports, structDefs)...)
		}
	}

	return structs, nil
}

// extractImports extracts import statements from an AST file and returns a map of package name -> import path
func extractImports(file *ast.File) map[string]string {
	imports := make(map[string]string)

	for _, imp := range file.Imports {
		// Get the import path (removing quotes)
		importPath := strings.Trim(imp.Path.Value, "\"")

		// Determine the package name
		var pkgName string
		if imp.Name != nil {
			// Named import (e.g., import foo "github.com/bar/foo")
			pkgName = imp.Name.Name
		} else {
			// Extract package name from import path (last segment)
			parts := strings.Split(importPath, "/")
			pkgName = parts[len(parts)-1]
		}

		imports[pkgName] = importPath
	}

	return imports
}

// discoverStructsInFile discovers structs in a single file
func discoverStructsInFile(fset *token.FileSet, fileName string, file *ast.File, pkgName string, fileImports map[string]string, structDefs map[string]*ast.StructType) []StructInfo {
	var structs []StructInfo

	// Analyze structs with embedded field resolution
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GenDecl:
			// Handle type declarations that may have comments
			for _, spec := range node.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if structType, ok := typeSpec.Type.(*ast.StructType); ok {
						// Comments are typically on the GenDecl, not the TypeSpec
						originalDoc := typeSpec.Doc
						if originalDoc == nil && node.Doc != nil {
							typeSpec.Doc = node.Doc
						}

						structInfo := analyzeStruct(fset, fileName, pkgName, typeSpec, structType, fileImports, structDefs)

						// Restore original doc to avoid side effects
						typeSpec.Doc = originalDoc

						if structInfo.HasEncxTags {
							structs = append(structs, structInfo)
						}
					}
				}
			}
		}
		return true
	})

	return structs
}

// analyzeStruct analyzes a struct type for encx tags
func analyzeStruct(fset *token.FileSet, fileName, pkgName string, typeSpec *ast.TypeSpec, structType *ast.StructType, fileImports map[string]string, structDefs map[string]*ast.StructType) StructInfo {
	structInfo := StructInfo{
		PackageName:       pkgName,
		StructName:        typeSpec.Name.Name,
		SourceFile:        filepath.Base(fileName),
		Fields:            []FieldInfo{},
		HasEncxTags:       false,
		GenerationOptions: make(map[string]string),
		RequiredImports:   make(map[string]string),
	}

	// Parse encx:options from struct-level comments
	if typeSpec.Doc != nil {
		parseEncxOptions(typeSpec.Doc, structInfo.GenerationOptions)
		// Validate generation options
		if err := validateGenerationOptions(structInfo.GenerationOptions); err != nil {
			// For now, we'll silently ignore invalid options to maintain compatibility
			// In the future, this could be made configurable
			_ = err
		}
	}

	// Analyze each field
	for _, field := range structType.Fields.List {
		// Handle embedded fields (anonymous/embedded structs)
		if len(field.Names) == 0 {
			// This is an embedded field
			embeddedFields := resolveEmbeddedField(field, fileImports, structDefs)
			for _, embeddedField := range embeddedFields {
				if len(embeddedField.EncxTags) > 0 {
					structInfo.HasEncxTags = true
				}
				structInfo.Fields = append(structInfo.Fields, embeddedField)

				// Track required imports from field types
				pkgNames := extractPackageNamesFromType(embeddedField.Type)
				for _, pkgName := range pkgNames {
					if importPath, found := fileImports[pkgName]; found {
						structInfo.RequiredImports[pkgName] = importPath
					}
				}
			}
			continue
		}

		// Handle regular named fields
		for _, name := range field.Names {
			fieldInfo := analyzeField(name.Name, field)
			if len(fieldInfo.EncxTags) > 0 {
				structInfo.HasEncxTags = true
			}
			structInfo.Fields = append(structInfo.Fields, fieldInfo)

			// Track required imports from field types
			pkgNames := extractPackageNamesFromType(fieldInfo.Type)
			for _, pkgName := range pkgNames {
				if importPath, found := fileImports[pkgName]; found {
					structInfo.RequiredImports[pkgName] = importPath
				}
			}
		}
	}

	// Note: Companion field validation removed - code generation creates separate structs

	return structInfo
}

// resolveEmbeddedField resolves an embedded struct field by looking up its definition
// and recursively extracting all its fields
func resolveEmbeddedField(field *ast.Field, fileImports map[string]string, structDefs map[string]*ast.StructType) []FieldInfo {
	var fields []FieldInfo

	// Get the embedded type name
	typeName := getTypeString(field.Type)

	// Handle pointer types (e.g., *DocumentBase)
	isPointer := false
	if strings.HasPrefix(typeName, "*") {
		isPointer = true
		typeName = strings.TrimPrefix(typeName, "*")
	}

	// Check if this is a local struct (no package prefix)
	if !strings.Contains(typeName, ".") {
		// Look up the struct definition in the current package
		if embeddedStructType, found := structDefs[typeName]; found {
			// Recursively process the embedded struct's fields
			for _, embeddedField := range embeddedStructType.Fields.List {
				// Handle nested embedded fields
				if len(embeddedField.Names) == 0 {
					nestedFields := resolveEmbeddedField(embeddedField, fileImports, structDefs)
					fields = append(fields, nestedFields...)
					continue
				}

				// Process regular fields from the embedded struct
				for _, name := range embeddedField.Names {
					fieldInfo := analyzeField(name.Name, embeddedField)

					// If the embedded struct was a pointer, the fields inherit that
					// (though in practice, field access syntax is the same)
					if isPointer && !strings.HasPrefix(fieldInfo.Type, "*") {
						// Note: We don't modify the type here because in Go,
						// whether a struct is embedded as pointer or value doesn't
						// change how you access its fields
					}

					fields = append(fields, fieldInfo)
				}
			}
		}
		// If not found, the embedded struct might be from another file in the package
		// but should have been collected in the first pass
	}
	// If it has a package prefix (e.g., other.Type), we can't resolve it
	// without parsing that external package, so we skip it

	return fields
}

// extractPackageNamesFromType extracts package identifiers from a type string
// e.g., "uuid.UUID" -> ["uuid"], "[]time.Time" -> ["time"], "*uuid.UUID" -> ["uuid"]
func extractPackageNamesFromType(typeStr string) []string {
	var packages []string
	seen := make(map[string]bool)

	// Remove array/slice/pointer prefixes
	cleanType := strings.TrimLeft(typeStr, "[]*")

	// Check if the type contains a package selector (.)
	if strings.Contains(cleanType, ".") {
		parts := strings.Split(cleanType, ".")
		if len(parts) >= 2 {
			pkgName := parts[0]
			if !seen[pkgName] {
				packages = append(packages, pkgName)
				seen[pkgName] = true
			}
		}
	}

	return packages
}

// analyzeField analyzes a single field for encx tags
func analyzeField(fieldName string, field *ast.Field) FieldInfo {
	fieldInfo := FieldInfo{
		Name:             fieldName,
		Type:             getTypeString(field.Type),
		EncxTags:         []string{},
		IsValid:          true,
		ValidationErrors: []string{},
	}

	// Extract encx tags from struct tags
	if field.Tag != nil {
		tagValue := strings.Trim(field.Tag.Value, "`")
		fieldInfo.EncxTags = extractEncxTags(tagValue)
	}

	// Validate tags if any exist
	if len(fieldInfo.EncxTags) > 0 {
		validator := NewTagValidator()
		errors := validator.ValidateFieldTags(fieldName, fieldInfo.EncxTags)
		if len(errors) > 0 {
			fieldInfo.IsValid = false
			fieldInfo.ValidationErrors = errors
		}
	}

	return fieldInfo
}

// getTypeString converts an ast.Expr to its string representation
func getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.ArrayType:
		return "[]" + getTypeString(t.Elt)
	case *ast.StarExpr:
		return "*" + getTypeString(t.X)
	case *ast.SelectorExpr:
		return getTypeString(t.X) + "." + t.Sel.Name
	default:
		return "unknown"
	}
}

// extractEncxTags extracts encx tags from a struct tag string
func extractEncxTags(tagString string) []string {
	// Simple implementation - in real code, use reflect.StructTag
	parts := strings.Split(tagString, " ")
	for _, part := range parts {
		// if strings.HasPrefix(part, "encx:") {
		if _, found := strings.CutPrefix(part, "encx:"); found {
			// Extract value between quotes
			value := strings.Trim(strings.TrimPrefix(part, "encx:"), "\"")
			return strings.Split(value, ",")
		}
	}
	return []string{}
}

// parseEncxOptions parses //encx:options comments and extracts key=value pairs
func parseEncxOptions(commentGroup *ast.CommentGroup, options map[string]string) {
	if commentGroup == nil {
		return
	}

	for _, comment := range commentGroup.List {
		text := strings.TrimSpace(comment.Text)

		// Remove comment prefixes
		if strings.HasPrefix(text, "//") {
			text = strings.TrimSpace(text[2:])
		} else if strings.HasPrefix(text, "/*") && strings.HasSuffix(text, "*/") {
			text = strings.TrimSpace(text[2 : len(text)-2])
		}

		// Check for encx:options prefix
		if strings.HasPrefix(text, "encx:options") {
			optionsText := strings.TrimSpace(text[12:]) // Remove "encx:options"
			parseOptionsPairs(optionsText, options)
		}
	}
}

// parseOptionsPairs parses key=value,key2=value2 format
func parseOptionsPairs(optionsText string, options map[string]string) {
	if optionsText == "" {
		return
	}

	pairs := strings.Split(optionsText, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" && value != "" {
				options[key] = value
			}
		}
	}
}

// validateGenerationOptions validates the generation options parsed from comments
func validateGenerationOptions(options map[string]string) error {
	for key := range options {
		switch key {
		case "serializer":
			return fmt.Errorf("serializer option is no longer supported; ENCX now uses a built-in compact serializer")
		default:
			// For now, ignore unknown options to allow for future extensions
			// Could be made stricter based on configuration
		}
	}
	return nil
}
