# docs/demos/screenshot-input.tape — QMP framebuffer screenshot + key input
Output yeast-screenshot.gif

Set Shell "bash"
Set FontSize 18
Set Width 1100
Set Height 700
Set Theme "Catppuccin Mocha"
Set Padding 24
Set TypingSpeed 60ms

Type "yeast screenshot web -o console.png"
Enter
Sleep 3s

Type "yeast input web --key ctrl_l+alt_l+f3"
Enter
Sleep 1.5s

Type "yeast input web --text 'root\\n'"
Enter
Sleep 2s

Type "yeast screenshot web -o tty3.png"
Enter
Sleep 3s

Type "ls -la *.png"
Enter
Sleep 2.5s
