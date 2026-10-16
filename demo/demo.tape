# demo.tape — record demo/demo.gif with vhs
#
#   brew install vhs ttyd ffmpeg
#   ./demo/fetch.sh            # downloads the demo book (public domain)
#   go build -o decoy .
#   vhs demo/demo.tape
#
# Demo text: 西遊記 by 吳承恩 (public domain), exported from Chinese Wikisource.
#
# The setup section is hidden with Hide/Show: it launches the app under a
# throwaway HOME (so the recording always starts at page one) and jumps into a
# chapter, skipping the Wikisource navigation templates that sit at the top of
# the title page and of each chapter.

Output demo/demo.gif

Set Shell bash
Set FontSize 16
Set FontFamily "Maple Mono CN"
Set Width 1000
Set Height 620
Set Padding 14
Set Theme "Dracula"
Set Framerate 20
Set TypingSpeed 55ms
Set CursorBlink false

Hide
Type "env HOME=/tmp/decoy-demo ./decoy demo/xiyouji.epub"
Enter
Sleep 2s

# Open the table of contents and jump to a later chapter.
Type "t"
Sleep 1s
Type "j"
Sleep 140ms
Type "j"
Sleep 140ms
Type "j"
Sleep 140ms
Type "j"
Sleep 140ms
Type "j"
Sleep 140ms
Type "j"
Sleep 140ms
Type "j"
Sleep 140ms
Type "j"
Sleep 140ms
Enter
Sleep 1.2s

# Scroll past the chapter-head navigation template.
Type "n"
Sleep 600ms
Type "n"
Sleep 900ms
Show
Sleep 800ms

# Page forward through the prose.
Type "n"
Sleep 800ms
Type "n"
Sleep 800ms

# Peek at the table of contents.
Type "t"
Sleep 1.6s
Type "Escape"
Sleep 800ms

# Boss key: the screen becomes a Claude Code session.
Type "b"
Sleep 3.5s

# Type into the fake prompt and get a fake reply.
Type "i"
Sleep 400ms
Type "fix the failing test"
Sleep 400ms
Enter
Sleep 3s

# Escape back to reading, then quit.
Type "Ctrl+B"
Sleep 1s
Type "q"
Sleep 800ms
