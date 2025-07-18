package objectstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
)

func TestSchemaStorageRepresentation(t *testing.T) {
	var baseJson = `
{
    "string_example": "Hello, world!",
    "integer": 12345,
    "float": 123.45,
    "boolean_true": true,
    "boolean_false": false,
    "null_value": null,
    
    "simple_object": {
        "key1": "value1",
        "key2": 2,
        "key3": false,
        "key4": null
    },

    "nested_objects": {
        "level1": {
            "level2": {
                "level3": {
                    "final_key": "deep_value"
                }
            }
        }
    },

    "array_of_strings": ["apple", "banana", "cherry"],
    
    "array_of_numbers": [1, 2, 3, 4.5, 6.78],
    
    "array_of_booleans": [true, false, true],

    "array_of_mixed_types": ["string", 123, true, null, {"inside_object_key": "inside_object_value"}],

    "array_of_objects": [
        {"name": "Alice", "age": 30, "is_student": false},
        {"name": "Bob", "age": 25, "is_student": true},
        {"name": "Charlie", "age": 35, "is_student": false}
    ],

    "multi_dimensional_array": [
        [1, 2, 3],
        [4, 5, 6],
        [7, 8, 9]
    ],
    
    "complex_mixed_array": [
        [
            {"sub_array_key1": "value1", "sub_array_key2": ["nested_array_item1", 2, 3.5, null]},
            {"sub_array_key3": true}
        ],
        123,
        "text"
    ],

    "deeply_nested_object": {
        "layer1": {
            "layer2": {
                "layer3": {
                    "layer4": {
                        "layer5": {
                            "layer6": {
                                "key": "deep_value"
                            }
                        }
                    }
                }
            }
        }
    },
    
    "empty_object": {},
    
    "empty_array": [],
    
    "object_with_empty_fields": {
        "empty_string": "",
        "empty_object": {},
        "empty_array": []
    },

    "complex_object_with_arrays": {
        "data": [
            {"id": 1, "values": [10, 20, 30]},
            {"id": 2, "values": [40, 50, {"nested_in_array": "complex_value"}]},
            {"id": 3, "values": []}
        ],
        "metadata": {
            "created_by": "user123",
            "tags": ["example", "test", "json"]
        }
    },

    "recursive_example": {
        "self_reference": {
            "self_reference": {
                "self_reference": {
                    "final_key": "end"
                }
            }
        }
    }
}
	`
	var equivalentJson = `
{
    "string_example": "Hello, world!",
    "integer": 12345,
    "float": 123.45,
    "boolean_true": true,
    "boolean_false": false,
    "null_value": null,
    
    "simple_object": {
        "key1": "value1",
        "key2": 2,
        "key3": false,
        "key4": null
    },

    "nested_objects": {
        "level1": {
            "level2": {
                "level3": {
                    "final_key": "deep_value"
                }
            }
        }
    },

    
    "array_of_numbers": [1, 2, 3, 4.5, 6.78],
    
    "array_of_booleans": [true, false, true],

    "array_of_mixed_types": ["string", 123, true, null, {"inside_object_key": "inside_object_value"}],

    "array_of_objects": [
        {"name": "Alice", "age": 30.0, "is_student": false},
        {"name": "Bob", "age": 25, "is_student": true},
        {"name": "Charlie", "age": 35, "is_student": false}
    ],

    "multi_dimensional_array": [
        [1, 2, 3],
        [4, 5, 6],
        [7, 8, 9]
    ],
    
    "complex_mixed_array": [
        [
            {"sub_array_key1": "value1", "sub_array_key2": ["nested_array_item1", 2, 3.5, null]},
            {"sub_array_key3": true}
        ],
        123,
        "text"
    ],

    "deeply_nested_object": {
        "layer1": {
            "layer2": {
                "layer3": {
                    "layer4": {
                        "layer5": {
                            "layer6": {
                                "key": "deep_value"
                            }
                        }
                    }
                }
            }
        }
    },
    
    "empty_object": {},

	"array_of_strings": ["apple", "banana", "cherry"],
    
    "object_with_empty_fields": {
        "empty_string": "",
        "empty_object": {},
        "empty_array": []
    },

    "complex_object_with_arrays": {
        "data": [
            {"id": 1, "values": [10, 20, 30]},
            {"id": 2, "values": [40, 50, {"nested_in_array": "complex_value"}]},
            {"id": 3.0, "values": []}
        ],
        "metadata": {
            "created_by": "user123",
            "tags": ["example", "test", "json"]
        }
    },

    "recursive_example": {
        "self_reference": {
            "self_reference": {
                "self_reference": {
                    "final_key": "end"
                }
            }
        }
    },

	"empty_array": []
}	
	`
	nonEqualJson := `
{
    "string_example": "Hello, world!",
    "integer": 12345,
    "float": 123.45,
    "boolean_true": true,
    "boolean_false": false,
    "null_value": null,
	"array_of_strings": ["apple", "cherry", "banana"],
    
    "simple_object": {
        "key1": "value1",
        "key2": 2,
        "key3": false,
        "key4": null
    },

    "nested_objects": {
        "level1": {
            "level2": {
                "level3": {
                    "final_key": "deep_value"
                }
            }
        }
    },

    
    "array_of_numbers": [1, 2, 3, 4.5, 6.78],
    
    "array_of_booleans": [true, false, true],

    "array_of_mixed_types": ["string", 123, true, null, {"inside_object_key": "inside_object_value"}],

    "array_of_objects": [
        {"name": "Alice", "age": 30, "is_student": false},
        {"name": "Bob", "age": 25, "is_student": true},
        {"name": "Charlie", "age": 35, "is_student": false}
    ],

    "multi_dimensional_array": [
        [1, 2, 3],
        [4, 5, 6],
        [7, 8, 9]
    ],
    
    "complex_mixed_array": [
        [
            {"sub_array_key1": "value1", "sub_array_key2": ["nested_array_item1", 2, 3.5, null]},
            {"sub_array_key3": true}
        ],
        123,
        "text"
    ],

    "deeply_nested_object": {
        "layer1": {
            "layer2": {
                "layer3": {
                    "layer4": {
                        "layer5": {
                            "layer6": {
                                "key": "deep_vale"
                            }
                        }
                    }
                }
            }
        }
    },
    
    "empty_object": {},

    
    "object_with_empty_fields": {
        "empty_string": "",
        "empty_object": {},
        "empty_array": []
    },

    "complex_object_with_arrays": {
        "data": [
            {"id": 1, "values": [10, 20, 30]},
            {"id": 2, "values": [40, 50, {"nested_in_array": "complex_value"}]},
            {"id": 3.0, "values": []}
        ],
        "metadata": {
            "created_by": "user123",
            "tags": ["example", "test", "json"]
        }
    },

    "recursive_example": {
        "self_reference": {
            "self_reference": {
                "self_reference": {
                    "final_key": "end"
                }
            }
        }
    },

	"empty_array": []
}	
	`
	var baseHash, equivalentHash string
	s := ObjectStorageRepresentation{
		Version: "0.1.0-alpha.1",
		Type:    catcommon.CatalogObjectTypeResource,
		Spec:    []byte(baseJson),
	}
	baseHash = s.GetHash()
	s.Spec = []byte(equivalentJson)
	equivalentHash = s.GetHash()
	assert.Equal(t, baseHash, equivalentHash)
	s.Spec = []byte(nonEqualJson)
	nonEqualHash := s.GetHash()
	assert.NotEqual(t, baseHash, nonEqualHash)
	sz, _ := s.Serialize()
	t.Logf("Serialized: %s", string(sz))
}
