# Goblade
[![Test](https://github.com/Sparta142/goblade/actions/workflows/test.yml/badge.svg)](https://github.com/Sparta142/goblade/actions/workflows/test.yml)
[![CodeQL](https://github.com/Sparta142/goblade/actions/workflows/codeql.yml/badge.svg)](https://github.com/Sparta142/goblade/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/sparta142/goblade)](https://goreportcard.com/report/github.com/sparta142/goblade)  
Lightweight, embeddable tool for capturing
[FINAL FANTASY XIV](https://www.finalfantasyxiv.com/) network traffic.

### Background
The possibilities offered by analyzing FINAL FANTASY XIV network traffic
are limitless. However, the current tools available for it can be slow and
difficult to work with. Goblade was developed to make the packet capture
and reassembly process as simple as possible for other developers.

## Usage

### Developers
1. Add Goblade to your project. In order of preference:
   1. Manually download it from the
      [Releases](https://github.com/Sparta142/goblade/releases) page and
      bundle it with your project.
   2. Add logic to download the latest build from GitHub at runtime.
   3. Build it yourself and do one of the previous two things.
2. Run it as a subprocess using the `live` subcommand, and capture its
   standard output (stdout).
   * There are other commands and options that can be used, however the
     default configuration is designed to just work in most cases.
3. Decode the output as [JSON Lines](https://jsonlines.org/).
   * The JSON schema is still in development.

### Players
Goblade is targeted towards developers of external tools. You'll only be able
to see "raw" data if you run it by itself.  
If you're still interested, you can download the latest version from the 
[Releases](https://github.com/Sparta142/goblade/releases) page and run it 
in a terminal (e.g., PowerShell, Command Prompt) like so:

```
./goblade.exe live
```

*Please be sure to use caution and follow all FINAL FANTASY XIV rules and 
policies when using Goblade or any other external tool.*

## Building
Goblade is only supported on Windows (x64). It can be provisionally built 
for other platforms (i.e., for testing purposes), but will not be able to 
handle [Oodle-compressed](http://www.radgametools.com/oodlenetwork.htm) data 
in such configurations.

### Prerequisites
* Go 1.19 or newer
* A pcap implementation (e.g., [Npcap](https://npcap.com/) on Windows,
  [libpcap-dev](https://packages.ubuntu.com/jammy/libpcap-dev) on Ubuntu)
* A cgo-compatible toolchain (e.g., [MinGW](https://www.mingw-w64.org/) on
  Windows)

### Build steps
```
git clone https://github.com/Sparta142/goblade
cd goblade
go generate ./...
go build .
```

### Reproducible Builds
Due to the difficulty of ensuring build determinism when using cgo, 
[Reproducible Builds](https://reproducible-builds.org/) are not yet supported.
This is a future goal.

## License
[MIT License](/LICENSE)

## Disclaimer
This project is not endorsed by, directly affiliated with, maintained,
authorized, or sponsored by Square Enix Holdings Co., Ltd., or any of its 
subsidiaries or its affiliates. All product and company names are the 
registered trademarks of their original owners. The use of any trade name or 
trademark is for identification and reference purposes only and does not imply 
any association with the trademark holder of their product brand.
