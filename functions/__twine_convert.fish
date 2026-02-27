function __twine_convert --description 'Internal: Convert regular repo to bare + worktree setup'
    # Show help
    if test "$argv[1]" = "--help"; or test "$argv[1]" = "-h"
        echo "convert - Convert regular repo to bare"
        echo ""
        echo "Usage: twine convert <repo>"
        echo ""
        echo "Arguments:"
        echo "  repo    Repository name (must exist in TWINE_BASE_DIRS)"
        echo ""
        echo "Examples:"
        echo "  twine convert my-project      # Convert existing repo"
        echo "  twine convert my-project   # Same thing"
        echo ""
        echo "What it does:"
        echo "  1. Clones regular repo as bare (my-project.git)"
        echo "  2. Creates worktree for current branch"
        echo "  3. Optionally removes old regular repo"
        echo ""
        echo "Before:"
        echo "  ~/src/work/my-project/     (regular repo)"
        echo ""
        echo "After:"
        echo "  ~/src/work/my-project.git/        (bare repo)"
        echo "  ~/src/work/my-project.git/main/   (worktree)"
        echo ""
        echo "Tab completion shows only regular (non-bare) repos"
        echo ""
        echo "Note: 'tw' will also offer to convert automatically"
        echo ""
        echo "See also: twine init, tw"
        return 0
    end

    if test (count $argv) -lt 1
        echo "Usage: twine convert <repo>"
        echo ""
        echo "Convert a regular git repository to bare repo + worktree setup"
        echo ""
        echo "Example:"
        echo "  twine convert my-project"
        echo ""
        echo "This will:"
        echo "  1. Clone the repo as bare (my-project.git)"
        echo "  2. Create worktree for current branch"
        echo "  3. Optionally remove the old repo"
        return 1
    end

    # Check for required configuration
    if not set -q TWINE_BASE_DIRS
        echo "Error: TWINE_BASE_DIRS not configured"
        echo "Add to ~/.config/fish/config.fish: set -gx TWINE_BASE_DIRS ~/src/repos"
        return 1
    end

    set repo $argv[1]
    set repo_path ""

    # Find the repo in base directories
    for base_dir in $TWINE_BASE_DIRS
        if test -d $base_dir/$repo
            set repo_path $base_dir/$repo
            break
        end
    end

    if test -z "$repo_path"
        echo "Error: Repository '$repo' not found in:"
        for base_dir in $TWINE_BASE_DIRS
            echo "  - $base_dir"
        end
        return 1
    end

    # Convert using helper function
    __tw_convert_to_bare $repo_path
end
