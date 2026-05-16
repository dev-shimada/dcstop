package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var logoLines = []string{
	` ██████╗   ██████╗ ███████╗ ████████╗  ██████╗  ██████╗ `,
	` ██╔══██╗ ██╔════╝ ██╔════╝ ╚══██╔══╝ ██╔═══██╗ ██╔══██╗`,
	` ██║  ██║ ██║      ███████╗    ██║    ██║   ██║ ██████╔╝ `,
	` ██║  ██║ ██║      ╚════██║    ██║    ██║   ██║ ██╔═══╝  `,
	` ██████╔╝ ╚██████╗ ███████║    ██║    ╚██████╔╝ ██║      `,
	` ╚═════╝   ╚═════╝ ╚══════╝    ╚═╝     ╚═════╝  ╚═╝      `,
}

// sunset gradient: orange → pink → purple
var sunsetColors = []lipgloss.Color{
	"#FF6B35",
	"#FF5062",
	"#FF4090",
	"#CC40C0",
	"#9940E0",
	"#6644F0",
}

// PrintLogo prints the dcstop logo with a sunset gradient.
func PrintLogo() {
	fmt.Println(LogoString())
}

// LogoString returns the dcstop logo as a sunset-gradient string.
func LogoString() string {
	lines := make([]string, len(logoLines))
	for i, line := range logoLines {
		color := sunsetColors[i%len(sunsetColors)]
		lines[i] = lipgloss.NewStyle().Foreground(color).Render(line)
	}
	return fmt.Sprintf("%s\n", lipgloss.JoinVertical(lipgloss.Left, lines...))
}
