function __tw_convert_to_bare --description 'Convert regular repo to bare + worktree setup'
    set repo_path $argv[1]

    if not test -d $repo_path
        echo "Error: Repository not found: $repo_path"
        return 1
    end

    # Check if already bare
    if string match -q "*.git" (basename $repo_path)
        echo "Repository is already bare: $repo_path"
        return 0
    end

    # Get repo name and current branch
    set repo_name (basename $repo_path)
    set base_dir (dirname $repo_path)

    cd $repo_path
    set current_branch (git rev-parse --abbrev-ref HEAD 2>/dev/null)

    if test -z "$current_branch"
        echo "Error: Not a git repository: $repo_path"
        return 1
    end

    echo "Converting $repo_name to bare repo + worktree setup..."
    echo "  Current branch: $current_branch"
    echo ""

    # Create bare repo
    set bare_path $base_dir/$repo_name.git

    if test -d $bare_path
        echo "Error: Bare repo already exists: $bare_path"
        return 1
    end

    echo "1. Cloning as bare repository..."
    if not git clone --bare $repo_path $bare_path
        echo "Error: Failed to clone as bare"
        return 1
    end
    echo "   ✓ Created: $bare_path"

    # Create worktree for current branch
    echo ""
    echo "2. Creating worktree for $current_branch..."
    if git -C $bare_path worktree add $current_branch $current_branch
        echo "   ✓ Created: $bare_path/$current_branch"
    else
        echo "   ⚠ Failed to create worktree, but bare repo exists"
    end

    # Ask about removing old repo
    echo ""
    echo "3. Old repository at: $repo_path"
    echo "   Remove it? (y/N)"
    read -P "   > " -n 1 remove_old

    if test "$remove_old" = "y"
        echo ""
        echo "   Removing old repository..."
        rm -rf $repo_path
        echo "   ✓ Removed: $repo_path"
    else
        echo ""
        echo "   ℹ Kept old repository (you can remove it manually)"
    end

    echo ""
    echo "✓ Conversion complete!"
    echo ""
    echo "New structure:"
    echo "  Bare repo:  $bare_path"
    echo "  Worktree:   $bare_path/$current_branch"
    echo ""
    echo "Use: tw $repo_name $current_branch"
end
