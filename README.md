# 🚀 PZAPP - Neural Network Port Scanner

<div align="center">

```
██████╗ ███████╗ █████╗ ██████╗ ██████╗ 
██╔══██╗╚══███╔╝██╔══██╗██╔══██╗██╔══██╗
██████╔╝  ███╔╝ ███████║██████╔╝██████╔╝
██╔═══╝  ███╔╝  ██╔══██║██╔═══╝ ██╔═══╝ 
██║     ███████╗██║  ██║██║     ██║     
╚═╝     ╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝     
```

**⚡ The Ultimate Cyberpunk Port Management Experience ⚡**

*A Matrix-inspired TUI for visualizing and terminating network ports with style*

</div>

---

## 🌌 Overview

PZAPP is a terminal-based port management tool that transforms the mundane task of port monitoring into an immersive cyberpunk experience. Built with Go and the Charm suite (Bubble Tea, Bubbles, Lip Gloss), it provides real-time visualization of active network ports with the ability to terminate processes through an epic, Matrix-inspired interface.

### ✨ Features

- 🎯 **Real-time Port Monitoring** - Live scanning of active network ports
- 💀 **Process Termination** - Safely kill processes with dramatic confirmation dialogs  
- 🔍 **Intelligent Search** - Filter ports by process name, protocol, or port number
- 🌈 **Dynamic Animations** - Matrix rain effects, glitch text, and pulsing colors
- 🎪 **Smart Port Classification** - Visual indicators for system, registered, and dynamic ports
- ⚡ **Protocol Detection** - Icons and states for TCP, UDP, HTTP, HTTPS connections
- 🎨 **Cyberpunk Aesthetic** - Full Matrix-inspired theme with neon colors and ASCII art

## 🛠️ Prerequisites

- **Go 1.24.2+** - Required for building the application
- **Unix-like OS** - macOS, Linux (uses `lsof` for port detection)
- **Terminal with Unicode support** - For proper rendering of special characters and emojis

## 📦 Installation & Building

### Clone and Build

```bash
# Clone the repository
git clone <repository-url>
cd pzapp

# Build the application
go build ./cmd/pzapp

# The binary will be created as 'pzapp' in the current directory
```

### Run Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...
```

## 🚀 Usage

### Basic Usage

```bash
# Run with live port detection (requires sudo/admin privileges)
./pzapp

# Alternative: run directly with go
go run ./cmd/pzapp
```

### Demo Mode

```bash
# Run with mock data for testing/demo purposes
PZAPP_USE_MOCK=1 ./pzapp

# Or with go run
PZAPP_USE_MOCK=1 go run ./cmd/pzapp
```

## 🎮 Controls

### 【 NAVIGATION PROTOCOLS 】
- `j/k` or `↑/↓` - Navigate through target list
- `enter` - Select current target for termination

### 【 COMBAT OPERATIONS 】  
- `d` or `enter` - Execute termination protocol on selected process
- `y/Y` - Confirm elimination in termination dialog
- `n/N/esc` - Abort current operation

### 【 SYSTEM OPERATIONS 】
- `r` - Reload target matrix (refresh port list)
- `/` - Initiate search protocol (filter ports)
- `esc` - Exit search mode
- `?` - Toggle command matrix (help screen)
- `q` or `ctrl+c` - Exit system

## 🎨 Interface Elements

### Port Visualization
Each port entry displays comprehensive information with visual indicators:

```
▶ 🔗 TCP ┃ 👑 80 ┃ 🎯 nginx [listening] ┃ 💀 1234 ┃ 👤 root ┃ 🌍 0.0.0.0:80
```

- **Protocol Icons**: 🔗 TCP, 📡 UDP, 🌐 HTTP, 🔐 HTTPS
- **Port Classification**: 👑 System (0-1023), 🎪 Registered (1024-49151), 🎲 Dynamic (49152+)
- **Connection States**: 🎯 Listening, 🔥 Established, ⏳ Close Wait, 💭 Time Wait
- **Process Info**: 💀 PID, 👤 User, 🌍 Address

### Dynamic Effects
- **Matrix Rain**: Animated digital rain effect at the top of the interface
- **Glitch Text**: Occasional text corruption effects for authentic cyberpunk feel
- **Color Cycling**: 6-color accent rotation every few animation ticks
- **Digital Noise**: Procedural noise patterns throughout the interface

### Epic Confirmations
When terminating a process, you'll see dramatic ASCII art warnings:

```
███████╗██╗    ██╗ █████╗ ██████╗ ███╗   ██╗██╗███╗   ██╗ ██████╗ 
██╔════╝██║    ██║██╔══██╗██╔══██╗████╗  ██║██║████╗  ██║██╔════╝ 
███████╗██║ █╗ ██║███████║██████╔╝██╔██╗ ██║██║██╔██╗ ██║██║  ███╗
╚════██║██║███╗██║██╔══██║██╔══██╗██║╚██╗██║██║██║╚██╗██║██║   ██║
███████║╚███╔███╔╝██║  ██║██║  ██║██║ ╚████║██║██║ ╚████║╚██████╔╝
╚══════╝ ╚══╝╚══╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝╚═╝  ╚═══╝ ╚═════╝
```

## 🏗️ Architecture

### Project Structure
```
pzapp/
├── cmd/pzapp/          # Application entry point
│   └── main.go         # CLI configuration and Bubble Tea program launch
├── internal/ui/        # TUI implementation
│   └── model.go        # Bubble Tea model, styles, and animations
├── internal/ports/     # Port detection and management
│   ├── provider.go     # Provider interface
│   ├── lsof.go         # Real port detection using lsof
│   ├── mock.go         # Mock provider for testing
│   └── termination.go  # Process termination logic
├── go.mod              # Go module definition
└── README.md           # This file
```

### Dependencies
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** - TUI framework
- **[Bubbles](https://github.com/charmbracelet/bubbles)** - TUI components (list)
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)** - Terminal styling

## 🔧 Development

### Building from Source
```bash
# Install dependencies
go mod download

# Build for current platform
go build ./cmd/pzapp

# Build for specific platform
GOOS=linux GOARCH=amd64 go build ./cmd/pzapp
```

### Code Structure
- **Provider Pattern**: Abstracted port detection allows for both real (`lsof`) and mock implementations
- **Bubble Tea Model**: Single model handles all UI state and interactions
- **Responsive Design**: Adaptive column widths and terminal resizing support
- **Animation System**: Tick-based animations with multiple timing cycles

### Testing
The application includes comprehensive tests for the port detection and parsing logic:

```bash
# Run unit tests
go test ./internal/ports/

# Run all tests with coverage
go test -cover ./...
```

## 🎭 Customization

### Color Themes
The cyberpunk color palette is defined in `internal/ui/model.go`. Key colors include:

- **Matrix Green**: `#00ff41` - Primary text and effects
- **Neon Cyan**: `#00ffff` - Accent highlights  
- **Hot Pink**: `#ff007f` - Secondary accents
- **Electric Gold**: `#ffd700` - Special highlights

### Animation Timing
Animation speeds can be adjusted by modifying the tick duration in `animationTickCmd()`:

```go
return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg {
    return tickMsg{when: t}
})
```

## 📋 Troubleshooting

### Common Issues

**"Permission denied" when running**
- PZAPP requires elevated privileges to access port information via `lsof`
- Run with `sudo ./pzapp` or use demo mode: `PZAPP_USE_MOCK=1 ./pzapp`

**"device not configured" error**  
- This occurs when running in non-TTY environments
- Use demo mode for testing: `PZAPP_USE_MOCK=1 ./pzapp`

**Unicode characters not displaying correctly**
- Ensure your terminal supports Unicode/UTF-8
- Try a modern terminal like iTerm2, Alacritty, or Windows Terminal

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit your changes: `git commit -m 'Add amazing feature'`
4. Push to the branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

### Development Guidelines
- Follow Go conventions and `gofmt` formatting
- Add tests for new functionality
- Update documentation for user-facing changes
- Maintain the cyberpunk aesthetic in UI changes

## 📜 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🌟 Acknowledgments

- **[Charm](https://charm.sh/)** - For the incredible TUI toolkit
- **Matrix Trilogy** - For the cyberpunk inspiration
- **TokyoNight Theme** - Original color palette inspiration (evolved into Matrix theme)

---

<div align="center">

**🚀 Welcome to the Matrix. Happy Port Hunting! 🚀**

*Built with ❤️ and ⚡ by cyberpunk enthusiasts*

</div>
