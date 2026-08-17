# Presenting the minimalistic Automatic-Mouse-Mover (AMM) App!

[![version][version-badge]][releases] [![Go Report Card](https://goreportcard.com/badge/github.com/samlara32/automatic-mouse-mover)](https://goreportcard.com/report/github.com/samlara32/automatic-mouse-mover) [![godoc-badge][godoc-badge]][godoc-link]

Ever felt the need to keep your machine awake without having to move the mouse pointer manually at regular intervals? **Well, not anymore!**

Introducing the simplest app that moves your mouse pointer at configurable intervals so that your machine stays awake. Best of all, it works **ONLY** when you are not actively using your computer, ensuring the cursor never moves unexpectedly while you work.

---

## ✨ Features & Enhancements

- 🍎 **Universal macOS Binary**: Runs natively on both **Apple Silicon (ARM64)** (M1/M2/M3/M4) and **Intel (x86_64)** without Rosetta emulation.
- ⏱ **Configurable Idle Intervals**: Choose check frequency: `30 Seconds`, `1 Minute (Default)`, `2 Minutes`, `5 Minutes`, or `10 Minutes`.
- 🎯 **Movement Modes**:
  - **Standard (10px)**: Default subtle shift back and forth.
  - **Micro-nudge (1px)**: Completely imperceptible shift to the human eye.
  - **Jiggle (Natural)**: Human-like randomized micro-motion.
- ⏳ **Auto-Stop Countdown Timers**: Set duration (`30m`, `1h`, `2h`, `4h`, or `Continuous`) so the app stops automatically when your task is done.
- 🎨 **Custom Tray Icons & Colors**: Choose between Mouse, Cloud, Man, and Geometric shapes with customizable colors (System default, Blue, Red, White).
- 🔒 **Automatic Code Signing**: Self-signs binaries during build to ensure seamless macOS Accessibility / TCC permissions.

---

## 📦 How to Install

### Option 1: Install from DMG / Binary

1. Download the latest `AutomaticMouseMover.dmg` or `amm.app.zip` from the [Releases](https://github.com/samlara32/automatic-mouse-mover/releases) page.
2. Drag `amm.app` into your `/Applications` folder.
3. Open `amm.app` from your Applications folder.

### Option 2: Build & Install from Source

Make sure you have `go` installed.

```bash
git clone https://github.com/samlara32/automatic-mouse-mover.git
cd automatic-mouse-mover

# Build Universal binary & DMG
make package

# Or build and directly install to /Applications
make install
```

---

## 🛡️ Granting Accessibility Permission on macOS

Because macOS protects input simulation, the first time you run AMM, you will need to grant Accessibility permission:

> Go to **System Settings** → **Privacy & Security** → **Accessibility** and toggle **`amm`** to **ON**.

---

## ⚙️ How It Works

Every selected interval (default: 60s), AMM checks system activity (keystrokes, mouse movement, screen changes, machine sleep). If no user activity was detected and the machine is awake, it gently shifts the cursor to keep the session active.

[version-badge]: https://img.shields.io/github/v/release/samlara32/automatic-mouse-mover.svg
[releases]: https://github.com/samlara32/automatic-mouse-mover/releases
[godoc-badge]: https://img.shields.io/badge/godoc-reference-blue.svg
[godoc-link]: https://godoc.org/github.com/samlara32/automatic-mouse-mover/pkg/mousemover
