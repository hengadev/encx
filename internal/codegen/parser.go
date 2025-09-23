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

		for fileName, file := range pkg.Files {
			structs = append(structs, discoverStructsInFile(fset, fileName, file, pkgName)...)
		}
	}

	return structs, nil
}

// discoverStructsInFile discovers structs in a single file
func discoverStructsInFile(fset *token.FileSet, fileName string, file *ast.File, pkgName string) []StructInfo {
	var structs []StructInfo

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

						structInfo := analyzeStruct(fset, fileName, pkgName, typeSpec, structType)

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
func analyzeStruct(fset *token.FileSet, fileName, pkgName string, typeSpec *ast.TypeSpec, structType *ast.StructType) StructInfo {
	structInfo := StructInfo{
		PackageName:       pkgName,
		StructName:        typeSpec.Name.Name,
		SourceFile:        filepath.Base(fileName),
		Fields:            []FieldInfo{},
		HasEncxTags:       false,
		GenerationOptions: make(map[string]string),
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
		for _, name := range field.Names {
			fieldInfo := analyzeField(name.Name, field)
			if len(fieldInfo.EncxTags) > 0 {
				structInfo.HasEncxTags = true
			}
			structInfo.Fields = append(structInfo.Fields, fieldInfo)
		}
	}

	// Note: Companion field validation removed - code generation creates separate structs

	return structInfo
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
