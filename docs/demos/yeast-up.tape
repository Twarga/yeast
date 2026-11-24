# docs/demos/yeast-up.tape — animated Pixel Yeast Cell boot flow
# Record:  vhs docs/demos/yeast-up.tape  (requires vhs + a KVM host)
Output yeast-up.gif

Set Shell "bash"
Set FontSize 18
Set Width 1100
Set Height 700
Set Theme "Catppuccin Mocha"
Set Padding 24
Set TypingSpeed 60ms

Type "yeast init"
Sleep 400ms
Enter
Sleep 1.5s

Type "yeast up"
Sleep 400ms
Enter
# The Pixel Yeast Cell animates through its boot stage here.
Sleep 8s

Type "yeast status"
Sleep 400ms
Enter
Sleep 2.5s
