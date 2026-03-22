Output "docs/demo.gif"

Set Width 1600
Set Height 1000
Set Shell "bash"

Type "ssh ssh.paullj.com"
Enter
Sleep 2s

# Navigate to second post (Cross-Compiling Rust)
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Enter
Sleep 1s

# Scroll down a few times
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms

# Hold down to scroll all the way down
Type@80ms "jjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjj"

# Hold down to scroll all the way down
Type@300ms "ddddddddddddddd"
Sleep 1s

# Go back
Type "q"
Sleep 1s

# Go to About tab
Type "	"
Sleep 1s

# Go to Posts tab
Type "	"
Sleep 1s

# Scroll down a few times
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "j"
Sleep 1s

# Exit
Type "q"
Sleep 1s
