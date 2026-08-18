# A demo of the overlay translating as you write. Render it with:
#   HERDR_PLUGIN_CONFIG_DIR="$(herdr plugin config-dir wazum.polyglot)" vhs demo/polyglot.tape
Output demo/polyglot.gif

Set Shell bash
Set FontSize 20
Set Width 1240
Set Height 430
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 55ms

# The launch is hidden, so the frame holds nothing but the overlay.
Hide
Type "clear && HERDR_POLYGLOT_TARGET=demo:p1 HERDR_POLYGLOT_LIVE=1 HERDR_POLYGLOT_VIM=0 HERDR_POLYGLOT_PROVIDER=deepl bin/herdr-polyglot"
Enter
Sleep 2s
Show

Sleep 2500ms
Type "Bitte behebe den fehlschlagenden Test in `berechneRabatt()`."
Sleep 3s
Type " Ergänze einen Fall für leere Warenkörbe."
Sleep 4s

Hide
Ctrl+C
