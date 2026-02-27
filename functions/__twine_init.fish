function __twine_init --description 'Internal: Clone a repository as bare + create initial worktree'
    # Show help
    if test "$argv[1]" = "--help"; or test "$argv[1]" = "-h"
        echo "init - Initialize bare repository"
        echo ""
        echo "Usage: twine init <repo-name> <git-url> [branch]"
        echo ""
        echo "Arguments:"
        echo "  repo-name    Name for the repository"
        echo "  git-url      Git URL to clone from"
        echo "  branch       Initial branch (optional, auto-detects default)"
        echo ""
        echo "Examples:"
        echo "  twine init my-project git@github.com:user/my-project.git"
        echo "  twine init my-project https://github.com/user/my-project.git main"
        echo "  twine init my-project git@github.com:user/my-project.git"
        echo ""
        echo "What it does:"
        echo "  1. Clones repository as bare (my-project.git)"
        echo "  2. Auto-detects default branch (main/master)"
        echo "  3. Creates worktree for initial branch"
        echo "  4. Ready to use with 'tw my-project'"
        echo ""
        echo "Creates:"
        echo "  \$TWINE_BASE_DIRS[1]/my-project.git/        (bare repo)"
        echo "  \$TWINE_BASE_DIRS[1]/my-project.git/main/   (worktree)"
        echo ""
        echo "See also: twine convert, tw"
        return 0
    end

    if test (count $argv) -lt 2
        echo "Usage: twine init <repo-name> <git-url> [branch]"
        echo ""
        echo "Clone a repository as bare and create initial worktree"
        echo ""
        echo "Arguments:"
        echo "  repo-name  Name for the repository (e.g., my-project)"
        echo "  git-url    Git URL to clone from"
        echo "  branch     Initial branch to checkout (default: default branch)"
        echo ""
        echo "Example:"
        echo "  twine init my-project git@github.com:user/my-project.git"
        echo "  twine init my-project git@github.com:user/my-project.git main"
        echo ""
        echo "This will create:"
        echo "  ~/src/repos/my-project.git/          (bare repo)"
        echo "  ~/src/repos/my-project.git/main/     (worktree)"
        return 1
    end

    # Check for required configuration
    if not set -q TWINE_BASE_DIRS
        echo "Error: TWINE_BASE_DIRS not configured"
        echo "Add to ~/.config/fish/config.fish: set -gx TWINE_BASE_DIRS ~/src/repos"
        return 1
    end

    set repo_name $argv[1]
    set git_url $argv[2]
    set initial_branch $argv[3]

    # Use first base directory for new repos
    set base_dir $TWINE_BASE_DIRS[1]
    set bare_path $base_dir/$repo_name.git

    if test -d $bare_path
        echo "Error: Repository already exists: $bare_path"
        return 1
    end

    echo "Initializing bare repository: $repo_name"
    echo "  Location: $bare_path"
    echo "  URL: $git_url"
    echo ""

    # Clone as bare
    echo "1. Cloning as bare repository..."
    if not git clone --bare $git_url $bare_path
        echo "   ✗ Failed to clone repository"
        return 1
    end
    echo "   ✓ Cloned: $bare_path"

    # Determine which branch to use
    if test -z "$initial_branch"
        # Get default branch from remote
        set initial_branch (git -C $bare_path symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@')

        if test -z "$initial_branch"
            # Fall back to common defaults
            if git -C $bare_path rev-parse --verify refs/remotes/origin/main >/dev/null 2>&1
                set initial_branch main
            else if git -C $bare_path rev-parse --verify refs/remotes/origin/master >/dev/null 2>&1
                set initial_branch master
            else
                echo "   ⚠ Could not determine default branch"
                echo "   Please specify branch: twine init $repo_name $git_url <branch>"
                return 1
            end
        end
    end

    echo ""
    echo "2. Creating worktree for $initial_branch..."

    # Create worktree
    if git -C $bare_path worktree add -b $initial_branch $initial_branch origin/$initial_branch
        echo "   ✓ Created: $bare_path/$initial_branch"
    else
        echo "   ⚠ Failed to create worktree"
        echo "   Bare repo exists, create worktree manually with:"
        echo "   git -C $bare_path worktree add <branch> <branch>"
        return 1
    end

    echo ""
    echo "✓ Repository initialized!"
    echo ""
    echo "Structure:"
    echo "  Bare repo:  $bare_path"
    echo "  Worktree:   $bare_path/$initial_branch"
    echo ""
    echo "Use: tw $repo_name $initial_branch"
end
