package tools

func RemoveDuplicateElement(s []string) []string {
	result := []string{}
	temp := map[string]struct{}{}
	for _, item := range s {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func CompareBytes(b1 []byte, b2 []byte) bool {
	// compare two byte array
	if len(b1) != len(b2) {
		return false
	}
	for i := range b1 {
		if b1[i] != b2[i] {
			return false
		}
	}
	return true
}
