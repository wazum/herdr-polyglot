# The overlay where it is used: a floating popup over an agent pane in a herdr
# session, recorded in a session of its own so nothing else is on screen.
#   bash demo/project.sh
#   herdr session stop polyglotdemo && herdr session delete polyglotdemo
#   vhs demo/incontext.tape
Output demo/polyglot-in-herdr.gif

Set Shell bash
Set FontSize 16
Set Width 1300
Set Height 720
Set Padding 16
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms

# Setup, before the first frame: an agent on the left, the project's tests on the
# right. ctrl+b is herdr's prefix.
Hide
Type "cd /tmp/polyglot-demo-project && clear && herdr --session polyglotdemo"
Enter
Sleep 6s
Type "herdr pane split --current --direction right --ratio 0.62 --focus --cwd /tmp/polyglot-demo-project >/dev/null"
Enter
Sleep 2s
Type "clear"
Enter
Type "go test ./..."
Enter
Sleep 4s
Ctrl+B
Type "h"
Sleep 1s
Type "clear && env $(env | grep -oE '^CLAUDE_[A-Z_]+' | sed 's/^/-u /' | tr '\\n' ' ') claude"
Enter
Sleep 12s
Enter
Sleep 8s
Show

# From here it is the plugin: the keybinding, the prompt, the translation, and the
# English typed into the agent's input.
Sleep 1500ms
Ctrl+B
Sleep 400ms
Type "t"
Sleep 2s
Type "Bitte behebe den fehlschlagenden Test im Warenkorb."
Sleep 3500ms
Type " Nutze Tabellentests."
Sleep 4s
Ctrl+R
Sleep 1500ms
Alt+Enter
Sleep 4s

Hide
