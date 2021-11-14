# Goblade
Lightweight, embeddable tool for capturing
[FINAL FANTASY XIV](https://www.finalfantasyxiv.com/) network traffic.

### Why does this exist?
The creative potential for analyzing (a.k.a. "parsing") FINAL FANTASY XIV
network traffic is huge. However, the current tools available for it are slow,
difficult to work with, and flimsy.

Goblade aims to make this kind of development as simple as possible. Here's how:

0. Distribute Goblade with your creation (or fetch the latest release from
   GitHub at runtime)
1. Run Goblade as a subprocess and capture standard output (stdout)
2. Decode Goblade's output as [JSON Lines](https://jsonlines.org/)
