package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEffectConfig_ParseSlides(t *testing.T) {
	tests := []struct {
		name        string
		slides      interface{}
		totalSlides int
		expected    []int
	}{
		{
			name:        "all slides",
			slides:      "all",
			totalSlides: 5,
			expected:    []int{0, 1, 2, 3, 4},
		},
		{
			name:        "specific slides as []interface{}",
			slides:      []interface{}{0, 2, 4},
			totalSlides: 10,
			expected:    []int{0, 2, 4},
		},
		{
			name:        "specific slides as []int",
			slides:      []int{1, 3, 5},
			totalSlides: 10,
			expected:    []int{1, 3, 5},
		},
		{
			name:        "empty interface{}",
			slides:      []interface{}{},
			totalSlides: 5,
			expected:    []int{},
		},
		{
			name:        "invalid type",
			slides:      123,
			totalSlides: 5,
			expected:    nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			effect := EffectConfig{
				Type:   "test-effect",
				Slides: tt.slides,
			}
			
			result := effect.ParseSlides(tt.totalSlides)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func FuzzEffectConfig_ParseSlides(f *testing.F) {
	// Add seed corpus
	f.Add(5)
	f.Add(10)
	f.Add(100)
	f.Add(0)
	
	f.Fuzz(func(t *testing.T, totalSlides int) {
		if totalSlides < 0 {
			return
		}
		
		effect := EffectConfig{
			Type:   "test",
			Slides: "all",
		}
		
		result := effect.ParseSlides(totalSlides)
		
		// Should return expected number of slides
		if totalSlides > 0 {
			assert.Len(t, result, totalSlides)
			
			// Should have correct values
			for i, slideIdx := range result {
				assert.Equal(t, i, slideIdx)
			}
		}
	})
}
