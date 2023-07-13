package util

func Int64ToInt32Pointer(val *int64) *int32 {
	if val == nil {
		return nil
	}
	x := int32(*val)
	return &x
}

func Int32ToInt64Pointer(val *int32) *int64 {
	if val == nil {
		return nil
	}
	x := int64(*val)
	return &x
}
