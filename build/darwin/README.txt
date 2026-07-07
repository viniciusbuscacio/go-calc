go-Calc for macOS
=================

Problem: The app does not open — "Apple could not verify go-Calc is free of malware."

Answer: run this command once in Terminal, then open the app normally:

    xattr -dr com.apple.quarantine ~/Downloads/go-calc.app


Project & source
----------------
https://github.com/viniciusbuscacio/go-calc
