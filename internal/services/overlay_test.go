package services

import (
	"testing"

	"gocreator/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestOverlayService_BuildTextOverlayFilterWithDuration(t *testing.T) {
	service := NewOverlayService()

	filter := service.BuildTextOverlayFilterWithDuration(config.EffectConfig{
		Type: "text-overlay",
		Config: config.EffectDetails{
			Text:              "Step 1: It's live",
			Position:          "top-left",
			OffsetX:           40,
			OffsetY:           24,
			Font:              "Arial",
			FontSize:          28,
			Color:             "yellow",
			OutlineColor:      "black",
			OutlineWidth:      2,
			BackgroundColor:   "black@0.5",
			BackgroundOpacity: 0.5,
			FadeIn:            0.5,
			FadeOut:           1.0,
		},
	}, 6)

	assert.Contains(t, filter, "drawtext=")
	assert.Contains(t, filter, "text='Step 1\\: It\\'s live'")
	assert.Contains(t, filter, "font='Arial'")
	assert.Contains(t, filter, "fontsize=28")
	assert.Contains(t, filter, "fontcolor=yellow")
	assert.Contains(t, filter, "borderw=2")
	assert.Contains(t, filter, "bordercolor=black")
	assert.Contains(t, filter, "box=1")
	assert.Contains(t, filter, "boxcolor=black@0.5")
	assert.Contains(t, filter, "boxborderw=5")
	assert.Contains(t, filter, "x=40")
	assert.Contains(t, filter, "y=24")
	assert.Contains(t, filter, "alpha='if(lt(t,0.500),t/0.500,if(lt(t,5.000),1,if(lt(t,6.000),(6.000-t)/1.000,0)))'")
}

func TestOverlayService_BuildTextOverlayFilter_DefaultPlacement(t *testing.T) {
	service := NewOverlayService()

	filter := service.BuildTextOverlayFilter(config.EffectConfig{
		Type:   "text-overlay",
		Config: config.EffectDetails{Text: "Hello"},
	})

	assert.Contains(t, filter, "text='Hello'")
	assert.Contains(t, filter, "fontcolor=white")
	assert.Contains(t, filter, "x=w-tw-10")
	assert.Contains(t, filter, "y=h-th-10")
	assert.NotContains(t, filter, "alpha=")
}

func TestOverlayService_BuildLogoOverlay(t *testing.T) {
	service := NewOverlayService()

	filter := service.BuildLogoOverlay("logo.png", "bottom-left", 0.5, 12, 16)

	assert.Equal(t, "movie=logo.png,format=rgba,colorchannelmixer=aa=0.50[logo];[in][logo]overlay=12:H-h-16", filter)
}
