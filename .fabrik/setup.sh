#!/bin/bash
# setup.sh
# Automates the setup of the Fabrik workflow framework.
# Installs battle-tested skills for OpenCode and initializes directories.

echo -e "🏭 \033[36mInitializing the Fabrik Method Setup...\033[0m"

# 1. Check Node.js / NPM dependency
if ! command -v node &> /dev/null || ! command -v npx &> /dev/null; then
    echo -e "\033[31mError: Node.js and NPX are required to install agent skills. Please install Node.js first.\033[0m"
    exit 1
fi

# 2. Check git initialization
if [ ! -d ".git" ]; then
    echo "Initializing git repository..."
    git init
fi

# 3. Create required directories
echo "Creating directory layout..."
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
mkdir -p "${SCRIPT_DIR}/.tasks/completed"
mkdir -p "${SCRIPT_DIR}/docs"
mkdir -p "${SCRIPT_DIR}/specs"
mkdir -p "${SCRIPT_DIR}/styleguide"
mkdir -p "${SCRIPT_DIR}/../src"

# Make loop files executable
chmod +x "${SCRIPT_DIR}/loop.sh"

# 4. Install mattpocock/skills using npx
echo -e "\033[33mInstalling battle-tested community skills from mattpocock/skills...\033[0m"
npx -y skills@latest add mattpocock/skills --agent opencode

# 5. Check if CONTEXT.md exists, if not create a default one
CONTEXT_FILE="${SCRIPT_DIR}/CONTEXT.md"
if [ ! -f "$CONTEXT_FILE" ]; then
    cat << 'EOF' > "$CONTEXT_FILE"
# Project Context & Dictionary

This document defines the domain terms and architecture rules for this project.

## Domain Language
*   **Term**: [Definition of specific project jargon to reduce agent verbosity]

## Architecture Guidelines
*   Standard imports should reference interfaces.
*   Prefer deep modules over thin helper wrappers.
EOF
fi

echo -e "\n\033[32m🎉 Setup Complete!\033[0m"
echo "Next steps:"
echo "1. Set D:/projects/fabrik as your active workspace."
echo "2. Trigger OpenCode and run: /setup-matt-pocock-skills"
echo "3. Start planning by running: ./.fabrik/loop.sh plan"
