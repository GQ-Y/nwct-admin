//go:build linux && !preview

package display

func newLinuxEvdevTouch(screenW, screenH int) touchReader {
	return newEvdevTouch(screenW, screenH)
}


