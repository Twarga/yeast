# docs/demos/fork.tape — clone a running VM into parallel copies
Output yeast-fork.gif

Set Shell "bash"
Set FontSize 18
Set Width 1100
Set Height 700
Set Theme "Catppuccin Mocha"
Set Padding 24
Set TypingSpeed 60ms

Type "yeast fork web --count 3 --boot"
Enter
# Copy-on-write siblings come up with fresh ports/MACs.
Sleep 6s

Type "yeast status"
Enter
Sleep 3s

Type "yeast destroy --forks web"
Enter
Sleep 3s
