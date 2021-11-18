# ffcat

Output per stream preview directly in terminal. Supports image, audio and video files.
Currently can only output via iTerm2 control codes.

**Be aware this is a quick proof of concept hack**

![ffcat demo](doc/demo.png)

## Requirements

Make sure you have a reasonably modern ffmpeg in `$PATH`.

## Install

```
# build and install latest master
GOPROXY=direct go install github.com/wader/ffcat@master
# copy binary to $PATH if needed
cp "$(go env GOPATH)/bin/ffcat" /usr/local/bin
```

## TODO and ideas

- Rename? is not really concatinating
- Combine wave form and spectragram?
- Silent/verbose output
- Pipe input, have to buffer?
- Timeline grid
- Seek from end support. -20: etc?
- Stats, loudness etc?
- Proper seek and frame select
- Select frames syntax?
- Render subtitles?
- Sixel output
- ANSI output
- PNG output if not a terminal
