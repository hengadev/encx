package processor

import (
	"context"
	"reflect"
)

// EmbeddedProcessor handles processing of embedded structs
type EmbeddedProcessor struct {
	structProcessor *StructProcessor
}

// NewEmbeddedProcessor creates a new EmbeddedProcessor instance
func NewEmbeddedProcessor(structProcessor *StructProcessor) *EmbeddedProcessor {
	return &EmbeddedProcessor{
		structProcessor: structProcessor,
	}
}

// ProcessEmbeddedStruct processes embedded structs within a parent struct
func (ep *EmbeddedProcessor) ProcessEmbeddedStruct(ctx context.Context, embeddedVal reflect.Value, embeddedType reflect.Type, errorCollector ErrorCollector) error {
	// Check if the embedded struct has encx tags
	if !hasEncxTags(embeddedType) {
		return nil // Skip if no encx tags found
	}

	// Create a pointer to the embedded struct for processing
	embeddedPtr := reflect.New(embeddedType)
	embeddedPtr.Elem().Set(embeddedVal)

	// Process the embedded struct
	err := ep.structProcessor.ProcessStruct(ctx, embeddedPtr.Interface(), errorCollector)
	if err != nil {
		return err
	}

	// Copy the processed values back to the original embedded struct
	embeddedVal.Set(embeddedPtr.Elem())

	return nil
}

// hasEncxTags checks if a struct type has any fields with encx tags
func hasEncxTags(structType reflect.Type) bool {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.Tag.Get("encx") != "" {
			return true
		}

		// Check embedded structs recursively
		if field.Type.Kind() == reflect.Struct {
			if hasEncxTags(field.Type) {
				return true
			}
		}
	}

	return false
}

// ProcessNestedStruct handles deeply nested structs with encx tags
func (ep *EmbeddedProcessor) ProcessNestedStruct(ctx context.Context, structVal reflect.Value, errorCollector ErrorCollector) error {
	structType := structVal.Type()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldVal := structVal.Field(i)

		// Skip unexported fields
		if !fieldVal.CanSet() {
			continue
		}

		// Process embedded structs
		if field.Type.Kind() == reflect.Struct {
			if err := ep.ProcessEmbeddedStruct(ctx, fieldVal, field.Type, errorCollector); err != nil {
				return err
			}
		}

		// Process pointer to structs
		if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct {
			if !fieldVal.IsNil() {
				if err := ep.ProcessEmbeddedStruct(ctx, fieldVal.Elem(), field.Type.Elem(), errorCollector); err != nil {
					return err
				}
			}
		}

		// Process slices of structs
		if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Struct {
			for j := 0; j < fieldVal.Len(); j++ {
				elemVal := fieldVal.Index(j)
				if err := ep.ProcessEmbeddedStruct(ctx, elemVal, field.Type.Elem(), errorCollector); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

