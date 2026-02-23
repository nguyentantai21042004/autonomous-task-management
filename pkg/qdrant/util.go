package qdrant

import (
	"fmt"

	pb "github.com/qdrant/go-client/qdrant"
)

// valueToInterface converts a qdrant Value to a Go interface{} (for payload extraction).
func valueToInterface(v *pb.Value) interface{} {
	if v == nil {
		return nil
	}
	switch v.GetKind().(type) {
	case *pb.Value_NullValue:
		return nil
	case *pb.Value_DoubleValue:
		return v.GetDoubleValue()
	case *pb.Value_IntegerValue:
		return v.GetIntegerValue()
	case *pb.Value_StringValue:
		return v.GetStringValue()
	case *pb.Value_BoolValue:
		return v.GetBoolValue()
	case *pb.Value_StructValue:
		st := v.GetStructValue()
		if st == nil {
			return nil
		}
		m := make(map[string]interface{})
		for key, val := range st.GetFields() {
			m[key] = valueToInterface(val)
		}
		return m
	case *pb.Value_ListValue:
		list := v.GetListValue()
		if list == nil {
			return nil
		}
		vals := list.GetValues()
		s := make([]interface{}, len(vals))
		for i, val := range vals {
			s[i] = valueToInterface(val)
		}
		return s
	default:
		return nil
	}
}

// GetDistanceMetric returns the appropriate distance metric for the given string (use Distance* constants).
func GetDistanceMetric(metric string) pb.Distance {
	switch metric {
	case DistanceCosine:
		return pb.Distance_Cosine
	case DistanceEuclidean:
		return pb.Distance_Euclid
	case DistanceDot:
		return pb.Distance_Dot
	case DistanceManhattan:
		return pb.Distance_Manhattan
	default:
		return pb.Distance_Cosine
	}
}

// ValidateVector checks if a vector is valid
func ValidateVector(vector []float32, expectedSize uint64) error {
	if len(vector) == 0 {
		return ErrInvalidVector
	}
	if uint64(len(vector)) != expectedSize {
		return fmt.Errorf("%w: expected size %d, got %d", ErrInvalidVector, expectedSize, len(vector))
	}
	return nil
}

// ValidateVectors validates multiple vectors
func ValidateVectors(vectors [][]float32, expectedSize uint64) error {
	if len(vectors) == 0 {
		return ErrInvalidVector
	}
	for i, vector := range vectors {
		if err := ValidateVector(vector, expectedSize); err != nil {
			return fmt.Errorf("invalid vector at index %d: %w", i, err)
		}
	}
	return nil
}
