# docs/demos/reset-superpowers.tape — live snapshot → break → ~instant restore
# The headline v1.4 demo: boot, snapshot live, destroy the guest fs, restore.
Output yeast-reset.gif

Set Shell "bash"
Set FontSize 18
Set Width 1100
Set Height 700
Set Theme "Catppuccin Mocha"
Set Padding 24
Set TypingSpeed 60ms

Type "yeast exec web -- touch /home/yeast/i-was-here"
Enter
Sleep 2s

Type "yeast snapshot web golden"
Enter
# Live checkpoint — the cell shows the working mood.
Sleep 4s

Type "yeast exec web -- rm -rf /home/yeast/i-was-here && touch /home/yeast/broken"
Enter
Sleep 2s

Type "yeast restore web golden"
Enter
# Back to golden in ~2s.
Sleep 4s

Type "yeast exec web -- ls /home/yeast/"
Enter
# i-was-here is back, broken is gone.
Sleep 3s
