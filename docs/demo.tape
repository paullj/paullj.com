Output "docs/demo.gif"

Set Width 1600
Set Height 1000
Set Shell "bash"

Type "ssh -o StrictHostKeyChecking=no ssh.paullj.com"
Enter
Sleep 5s

# Navigate to a recent post on the home page
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Enter
Sleep 2s

# Scroll down in the post
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 1s

# Go back
Type "q"
Sleep 1s

# Go to About tab
Type "	"
Sleep 3s

# Go to Posts tab
Type "	"
Sleep 3s

# Exit
Type "q"
Sleep 1s
