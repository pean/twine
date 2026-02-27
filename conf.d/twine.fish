# Twine plugin initialization and configuration validation

# Validate required configuration
if not set -q TWINE_BASE_DIRS
    echo "⚠️  Twine: TWINE_BASE_DIRS not configured"
    echo "   Add to ~/.config/fish/config.fish:"
    echo "   set -gx TWINE_BASE_DIRS ~/src/repos"
end

# Set defaults for optional configuration
if not set -q TWINE_SESSION_PREFIX
    set -gx TWINE_SESSION_PREFIX ""
end

if not set -q TWINE_USE_TMUXINATOR
    set -gx TWINE_USE_TMUXINATOR auto
end

# Helper function to check if tmuxinator should be used
function __twine_use_tmuxinator
    # Explicit disable
    if test "$TWINE_USE_TMUXINATOR" = "false"
        return 1
    end

    # Check if tmuxinator is installed
    if not command -v tmuxinator >/dev/null
        return 1
    end

    # Explicit enable
    if test "$TWINE_USE_TMUXINATOR" = "true"
        return 0
    end

    # Auto mode: use if tmuxinator exists and layout is configured
    if set -q TWINE_TMUXINATOR_LAYOUT
        return 0
    end

    return 1
end
