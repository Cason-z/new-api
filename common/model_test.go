package common

import "testing"

func TestIsImageGenerationModelRecognizesMAIImage(t *testing.T) {
	if !IsImageGenerationModel("MAI-Image-2.5") {
		t.Fatal("expected MAI-Image-2.5 to be recognized as an image generation model")
	}
}
