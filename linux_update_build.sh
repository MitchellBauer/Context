#!/bin/bash

# --- Configuration ---
PROJECT_DIR="$HOME/Projects/context"
BINARY_NAME="context"
INSTALL_DIR="$HOME/.local/bin"

# Navigate to project directory
cd "$PROJECT_DIR" || { echo "Directory not found! Exiting."; exit 1; }

# ensure install dir exists
mkdir -p "$INSTALL_DIR"

# --- SAFETY CHECK (New) ---
# If I have local changes, DO NOT auto-update.
if [ -n "$(git status --porcelain)" ]; then
    echo "Local changes detected (Dev Mode). Skipping auto-update."
    exit 0
fi

# --- Logic Check ---
NEEDS_BUILD=false

# Condition 1: Does the binary exist?
if [ ! -f "$BINARY_NAME" ]; then
    echo "Binary missing. Build required."
    NEEDS_BUILD=true
fi

# Condition 2: Are there updates on GitHub?
git fetch origin
LOCAL=$(git rev-parse HEAD)
REMOTE=$(git rev-parse @{u})

if [ "$LOCAL" != "$REMOTE" ]; then
    echo "Update found. Pulling changes..."
    git pull
    NEEDS_BUILD=true
else
    echo "Git repository is up to date."
fi

# --- Build & Install ---
if [ "$NEEDS_BUILD" = true ]; then
    echo "Building $BINARY_NAME..."
    go build -o "$BINARY_NAME"

    if [ $? -eq 0 ]; then
        echo "Build successful."

        # Install logic (Fixed for Symlink issue)
        echo "Installing to $INSTALL_DIR..."
        rm -f "$INSTALL_DIR/$BINARY_NAME"
        cp "$BINARY_NAME" "$INSTALL_DIR/"

        echo "Success! version updated."
    else
        echo "Build failed."
        exit 1
    fi
else
    echo "No build necessary."
fi
