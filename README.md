# Pssh

<img width="768" height="112" alt="Image" src="https://github.com/user-attachments/assets/94fd8363-7ada-4294-a663-69bac18e4e91" />

A simple TUI using [bubbletea](https://github.com/charmbracelet/bubbletea) wrapper around ssh that launches a searchable menu of hosts loaded from the `.ssh/config` file.

Hosts are grouped by their name if they share similar hostnames using a first in best dressed approach.

## Installation

You can locally install this with:

```bash
go install github.com/pix-xip/pssh@latest 
```
