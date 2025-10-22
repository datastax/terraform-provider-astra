package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

func Coalesce[T any](values ...*T) *T {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func Elvis[T any](value *T, defaultValue T) T {
	if value != nil {
		return *value
	}
	return defaultValue
}

func MergeMaps[K comparable, V any](maps ...map[K]V) map[K]V {
	result := make(map[K]V)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

func StringEnumPtrToStrPtr[E ~string](value *E) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(string(*value))
}

func IntPtrToTypeInt32Ptr(value *int) types.Int32 {
	if value == nil {
		return types.Int32Null()
	}
	return types.Int32Value(int32(*value))
}
