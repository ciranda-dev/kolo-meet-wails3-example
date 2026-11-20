# Notch and tray

The **notch** is the black cut-out at the top of some Mac screens, around the camera. The example shows a call bar in a small window at the top of the screen, around the notch when there is one. The **tray** is the small icon area of the system: the menu bar on macOS, the notification area on Windows, the tray on Linux.

## Tray (macOS, Windows, Linux)

- The menu has Open Kolo Meet, the status line, Copy invite for the host, the microphone and camera items, Leave the call while in one, the notch setting (macOS only), and Quit.
- Clicking the icon shows or hides the window; the example keeps running while the window is hidden.
- Closing the window quits the example.
- On Linux the tray is a D-Bus service: in a session without a D-Bus session bus the example runs without a tray icon and logs one line.

## Notch window (macOS only)

On a Mac, a second window made with Wails' `NewNotchWindow` shows the call bar: the number of people, who is speaking, the microphone and camera buttons, the time, Leave, and a Copy invite button for the host.

- It hangs from the top of the screen on every Mac; on a Mac with a notch, its shape follows the notch.
- It is hidden at start, opens when a call becomes active, and hides 6 seconds after it ends.
- Cmd+Alt+Shift+M shows it again. This keyboard shortcut exists on macOS only.
- On Windows and Linux the notch window does nothing.

## Turn it off

"Show calls in the notch", in Settings or in the menu bar menu, turns the notch window off.

## What reaches the notch

The invite never reaches the notch page: it gets `hasInvite`, and `NotchCopyInvite` copies the invite in Go.

## Build tag

The example is built with public macOS APIs, so the notch page is opaque instead of transparent.
