package event

// isLetter 判断是否是字母
func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isDigit 判断是否是数字
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// isValidFieldName 验证字段名格式（只允许字母、数字、下划线，且不能以数字开头）
func isValidFieldName(fieldName string) bool {
	if len(fieldName) == 0 {
		return false
	}

	// 第一个字符必须是字母或下划线
	if !isLetter(fieldName[0]) && fieldName[0] != '_' {
		return false
	}

	// 后续字符可以是字母、数字或下划线
	for i := 1; i < len(fieldName); i++ {
		c := fieldName[i]
		if !isLetter(c) && !isDigit(c) && c != '_' {
			return false
		}
	}

	return true
}
