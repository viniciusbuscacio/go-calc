go-Calc for macOS
=================

go-Calc is not notarized by Apple (notarization requires a paid Apple
Developer account). Because of that, the first time you open a downloaded
copy, macOS shows a warning like:

    "Apple could not verify go-Calc is free of malware."

The app is safe — this is just Gatekeeper being cautious about apps from
outside the App Store. You only have to approve it ONCE.

How to open it (one time)
-------------------------
1. Move  go-calc.app  to your  Applications  folder (recommended).
2. Double-click it. When the warning appears, click  Done  /  Cancel.
3. Open   menu (top-left)  →  System Settings  →  Privacy & Security.
4. Scroll down to the message about "go-Calc" and click  Open Anyway.
5. Confirm. From now on it opens normally, with no warning.

Faster way (Terminal)
---------------------
Run this once, then just open the app:

    xattr -dr com.apple.quarantine /Applications/go-calc.app

(adjust the path if the app is somewhere other than Applications)

Project & source
----------------
https://github.com/viniciusbuscacio/go-calc
