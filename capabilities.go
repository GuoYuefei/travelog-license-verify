package verify

// GetCapability safely retrieves a capability value by key from a capabilities map.
// Returns the value and whether the key exists.
func GetCapability(caps map[string]any, key string) (any, bool) {
	if caps == nil {
		return nil, false
	}
	v, ok := caps[key]
	return v, ok
}

// GetCapabilityString retrieves a string capability value.
// Returns the value and whether the key exists and is a string.
func GetCapabilityString(caps map[string]any, key string) (string, bool) {
	v, ok := GetCapability(caps, key)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// GetCapabilityInt retrieves an int capability value.
// JSON numbers are typically decoded as float64, so this handles both float64 and int.
func GetCapabilityInt(caps map[string]any, key string) (int, bool) {
	v, ok := GetCapability(caps, key)
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}

// GetCapabilityBool retrieves a bool capability value.
func GetCapabilityBool(caps map[string]any, key string) (bool, bool) {
	v, ok := GetCapability(caps, key)
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}
