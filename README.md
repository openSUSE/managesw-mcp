# managesw-mcp

`managesw-mcp` is a server that exposes OS software management functions through the Model Context Protocol (MCP). It allows you to manage packages, repositories, and patches on a Linux system.
Most of the functions are only available on `zypper` and `dnf` based systems 

## Installation from the source

1.  **Install Go:** Make sure you have Go version 1.22.x or later installed.
2.  **Clone the repository:**
    ```bash
    git clone https://github.com/openSUSE/managesw-mcp.git
    cd managesw-mcp
    ```
3.  **Build the server:**
    ```bash
    make
    ```
    or
    ```bash
    go build
    ```

## Usage

To interact with the `managesw-mcp` server, you can use `mcptools`, a command-line interface for MCP servers. You can find more information about `mcptools` at [https://github.com/f/mcptools](https://github.com/f/mcptools).

A simple call would be:
```
  ~/go/bin/mcptools call list_packages go run managesw-mcp.go  
```
