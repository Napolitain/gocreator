package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultTransitionConfig(t *testing.T) {
	config := DefaultTransitionConfig()
	
	assert.Equal(t, TransitionFade, config.Type)
	assert.Equal(t, 0.5, config.Duration)
}

func TestTransitionConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  TransitionConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid fade transition",
			config: TransitionConfig{
				Type:     TransitionFade,
				Duration: 0.5,
			},
			wantErr: false,
		},
		{
			name: "valid wipe transition",
			config: TransitionConfig{
				Type:     TransitionWipeleft,
				Duration: 1.0,
			},
			wantErr: false,
		},
		{
			name: "valid slide transition",
			config: TransitionConfig{
				Type:     TransitionSlideright,
				Duration: 0.75,
			},
			wantErr: false,
		},
		{
			name: "valid none transition",
			config: TransitionConfig{
				Type:     TransitionNone,
				Duration: 0.0,
			},
			wantErr: false,
		},
		{
			name: "negative duration",
			config: TransitionConfig{
				Type:     TransitionFade,
				Duration: -0.5,
			},
			wantErr: true,
			errMsg:  "transition duration must be non-negative",
		},
		{
			name: "duration too long",
			config: TransitionConfig{
				Type:     TransitionFade,
				Duration: 6.0,
			},
			wantErr: true,
			errMsg:  "transition duration is too long",
		},
		{
			name: "invalid transition type",
			config: TransitionConfig{
				Type:     "invalid",
				Duration: 0.5,
			},
			wantErr: true,
			errMsg:  "invalid transition type",
		},
		{
			name: "zero duration is valid",
			config: TransitionConfig{
				Type:     TransitionFade,
				Duration: 0.0,
			},
			wantErr: false,
		},
		{
			name: "max duration is valid",
			config: TransitionConfig{
				Type:     TransitionFade,
				Duration: 5.0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTransitionConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		config  TransitionConfig
		enabled bool
	}{
		{
			name: "fade transition enabled",
			config: TransitionConfig{
				Type:     TransitionFade,
				Duration: 0.5,
			},
			enabled: true,
		},
		{
			name: "none transition disabled",
			config: TransitionConfig{
				Type:     TransitionNone,
				Duration: 0.5,
			},
			enabled: false,
		},
		{
			name: "zero duration disabled",
			config: TransitionConfig{
				Type:     TransitionFade,
				Duration: 0.0,
			},
			enabled: false,
		},
		{
			name: "wipe transition enabled",
			config: TransitionConfig{
				Type:     TransitionWipeleft,
				Duration: 1.0,
			},
			enabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.enabled, tt.config.IsEnabled())
		})
	}
}

func TestTransitionConfig_GetFFmpegTransitionName(t *testing.T) {
	tests := []struct {
		name           string
		transitionType TransitionType
		expectedName   string
	}{
		{
			name:           "fade",
			transitionType: TransitionFade,
			expectedName:   "fade",
		},
		{
			name:           "wipeleft",
			transitionType: TransitionWipeleft,
			expectedName:   "wipeleft",
		},
		{
			name:           "wiperight",
			transitionType: TransitionWiperight,
			expectedName:   "wiperight",
		},
		{
			name:           "wipeup",
			transitionType: TransitionWipeup,
			expectedName:   "wipeup",
		},
		{
			name:           "wipedown",
			transitionType: TransitionWipedown,
			expectedName:   "wipedown",
		},
		{
			name:           "slideleft",
			transitionType: TransitionSlideleft,
			expectedName:   "slideleft",
		},
		{
			name:           "slideright",
			transitionType: TransitionSlideright,
			expectedName:   "slideright",
		},
		{
			name:           "slideup",
			transitionType: TransitionSlideup,
			expectedName:   "slideup",
		},
		{
			name:           "slidedown",
			transitionType: TransitionSlidedown,
			expectedName:   "slidedown",
		},
		{
			name:           "dissolve maps to fade",
			transitionType: TransitionDissolve,
			expectedName:   "fade",
		},
		{
			name:           "none returns empty",
			transitionType: TransitionNone,
			expectedName:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := TransitionConfig{Type: tt.transitionType}
			assert.Equal(t, tt.expectedName, config.GetFFmpegTransitionName())
		})
	}
}
