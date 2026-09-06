package main

import (
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sarkarshuvojit/bubblebook/pkg/bubblebook"
)

var registerBubblebookStory storyRegistrar = func(name string, factory func() tea.Model) {
	bubblebook.Register(name, bubblebook.ComponentFactory(factory))
}

var startBubblebook = bubblebook.Start
var bubblebookOutput io.Writer = os.Stdout

func main() {
	registerStories(registerBubblebookStory, colorModeForOutput(bubblebookOutput))
	startBubblebook()
}
