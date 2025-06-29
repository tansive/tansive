package errors

func ErrMissingRequiredAttribute(attr string, value ...any) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "missing required attribute",
	}
}

func ErrInvalidStepValue(attr string, value ...any) ValidationError {
	return ValidationError{
		Field: attr,
		Value: value,
		ErrStr: "step value is invalid: must be non-zero and must be in sequence with maxValue and minValue; " +
			"if positive, minValue must be present; if negative, maxValue must be present",
	}
}

func ErrMissingSchemaOrType(attr string, value ...any) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "should contain either schema or dataType",
	}
}

func ErrShouldContainSchemaOrType(attr string, value ...any) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "should contain either schema or dataType",
	}
}

func ErrMaxValueLessThanMinValue(attr string, value ...any) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "maxValue must be greater than minValue",
	}
}

func ErrValidationFailed(attr string, value ...any) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "validation failed",
	}
}

func ErrInvalidFieldSchema(attr string, value ...any) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "invalid schema",
	}
}

func ErrInvalidNameFormat(attr string, value ...string) ValidationError {
	var errStr string
	if len(value) == 0 {
		errStr = "invalid name format; allowed characters: [a-z0-9-]"
	} else {
		errStr = "invalid name format " + InQuotes(value[0]) + "; allowed characters: [a-z0-9-]"
	}
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: errStr,
	}
}

func ErrInvalidObjectPath(attr string, value ...any) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "invalid resource path; must start with '/' and contain only alphanumeric characters, underscores, and hyphens",
	}
}

func ErrInvalidCatalogVersion(attr string, value ...any) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "invalid catalog version",
	}
}

func ErrInvalidVersion(attr string, value ...any) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "invalid version",
	}
}

func ErrInvalidResourceURI(attr string, value ...any) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "invalid resource uri",
	}
}

func ErrInvalidViewRuleIntent(attr string, value ...any) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "invalid view rule intent",
	}
}

func ErrInvalidViewRuleAction(attr string, value ...any) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "invalid view rule action",
	}
}

func ErrInvalidAnnotation(attr string, value ...string) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "annotation keys cannot contain spaces",
	}
}

func ErrUnsupportedKind(attr string, value ...string) ValidationError {
	var errStr string
	if len(value) == 0 {
		errStr = "unsupported kind"
	} else {
		errStr = "unsupported kind " + InQuotes(value[0])
	}
	return ValidationError{
		Field:  "",
		Value:  value,
		ErrStr: errStr,
	}
}

func ErrUnsupportedDataType(attr string, value ...string) ValidationError {
	var errStr string
	if len(value) == 0 {
		errStr = "invalid data type for version"
	} else {
		errStr = "invalid data type " + InQuotes(value[0]) + " for version"
	}
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: errStr,
	}
}

func ErrUndefinedParameterSchema(attr string, value ...any) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "undefined parameter schema",
	}
}

func ErrMissingObjectLoaders(attr string, value ...any) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "unable to resolve path; missing object loaders",
	}
}

func ErrParameterSchemaDoesNotExist(attr string, value ...any) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "parameter schema does not exist",
	}
}

func ErrInvalidValue(attr string, value ...any) ValidationError {
	errStr := "invalid value"
	if len(value) > 0 {
		if str, ok := value[0].(string); ok {
			errStr = str
		}
	}
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: errStr,
	}
}

func ErrInvalidParameter(attr string, value ...any) ValidationError {
	return ValidationError{
		Field:  attr,
		Value:  value,
		ErrStr: "invalid parameter",
	}
}
