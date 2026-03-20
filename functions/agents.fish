function agents -d "List and switch to AI coding agents (Claude Code & OpenCode) in tmux"
    if not command -q fzf
        echo "Error: fzf is required for interactive selection" >&2
        return 1
    end

    if not command -q tmux
        echo "Error: tmux is not installed" >&2
        return 1
    end

    # Build list of agents with metadata
    set -l agent_list

    # Find Claude Code sessions
    for session_file in ~/.claude/sessions/*.json
        if not test -f $session_file
            continue
        end

        set -l pid (basename $session_file .json)

        # Claude's PID is a child of the shell running in the tmux pane.
        # Walk up the process tree to find the ancestor that matches a pane_pid.
        set -l pane_pid $pid
        for i in (seq 5)
            set -l ppid (ps -p $pane_pid -o ppid= 2>/dev/null | string trim)
            if test -z "$ppid" -o "$ppid" = "1" -o "$ppid" = "0"
                set pane_pid ""
                break
            end
            set pane_pid $ppid
            if tmux list-panes -a -F "#{pane_pid}" 2>/dev/null | grep -q "^$pane_pid\$"
                break
            end
        end

        if test -z "$pane_pid"
            continue
        end

        # Find tmux pane for this shell PID
        set -l tmux_info (tmux list-panes -a -F "#{pane_pid} #{session_name} #{pane_current_path} #{pane_id}" 2>/dev/null | grep "^$pane_pid ")

        if test -n "$tmux_info"
            set -l parts (string split ' ' $tmux_info)
            set -l session_name $parts[2]
            set -l current_path $parts[3]
            set -l pane_id $parts[4]

            # Parse session JSON for additional info
            set -l model "unknown"
            set -l dir $current_path

            # Try to extract model and directory from JSON
            if command -q jq
                set model (jq -r '.model // "unknown"' $session_file 2>/dev/null)
                set -l json_dir (jq -r '.directory // ""' $session_file 2>/dev/null)
                if test -n "$json_dir"
                    set dir $json_dir
                end
            else
                # Fallback: basic grep parsing if jq not available
                set -l model_line (grep -o '"model":"[^"]*"' $session_file 2>/dev/null)
                if test -n "$model_line"
                    set model (string replace -r '.*"model":"([^"]*)".*' '$1' $model_line)
                end
            end

            # Detect state from pane content
            set -l state (__agents_detect_claude_state $pane_id)
            set -l colored_state (__agents_colorize_state $state)

            # Shorten directory path for display
            set -l short_dir (string replace -r "^$HOME" "~" $dir)

            # Format: indicator | type | state | session | directory | model
            set -a agent_list "🤖|Claude Code|$colored_state|$session_name|$short_dir|$model"
        end
    end

    # Find OpenCode processes in tmux
    set -l opencode_panes (tmux list-panes -a -F "#{pane_pid} #{session_name} #{pane_current_path} #{pane_current_command} #{pane_id}" 2>/dev/null | grep "opencode\$")

    for line in $opencode_panes
        set -l parts (string split ' ' $line)
        set -l pid $parts[1]
        set -l session_name $parts[2]
        set -l current_path $parts[3]
        set -l pane_id $parts[5]

        # Detect state from pane content
        set -l state (__agents_detect_opencode_state $pane_id)
        set -l colored_state (__agents_colorize_state $state)

        # Shorten directory path for display
        set -l short_dir (string replace -r "^$HOME" "~" $current_path)

        # Format: indicator | type | state | session | directory | model
        set -a agent_list "🔓|OpenCode|$colored_state|$session_name|$short_dir|-"
    end

    # Check if we found any agents
    if test (count $agent_list) -eq 0
        echo "No AI coding agents found running in tmux sessions" >&2
        return 1
    end

    # Display with fzf (use full height if in popup, otherwise 40%)
    set -l fzf_height "40%"
    if test -n "$TMUX_POPUP"
        set fzf_height "100%"
    end

    set -l selected (printf '%s\n' $agent_list | \
        column -t -s '|' | \
        fzf --ansi \
            --height $fzf_height \
            --reverse \
            --border \
            --header "AI Coding Agents - Select to switch (Esc to cancel)" \
            --preview 'echo {}' \
            --preview-window hidden \
            --prompt "Agent > ")

    if test -z "$selected"
        return 0
    end

    # Extract session name from selection (4th column now, due to state)
    set -l session_name (echo $selected | awk '{print $4}')

    # Switch to the selected session
    if set -q TMUX
        tmux switch-client -t $session_name
    else
        tmux attach -t $session_name
    end
end

function __agents_detect_claude_state -a pane_id -d "Detect Claude Code agent state from pane content"
    # Capture last few lines of pane (where status bar typically is)
    set -l pane_content (tmux capture-pane -p -t $pane_id -S -10 2>/dev/null)

    # Check for state indicators in Claude Code's TUI
    if string match -q "*esc to interrupt*" -- $pane_content
        echo "⚡Working"
    else if string match -q "*Esc to cancel*" -- $pane_content
        echo "❗Input"
    else if string match -q "*Enter to submit*" -- $pane_content
        echo "⏸ Idle"
    else if string match -q "*Welcome to Claude Code*" -- $pane_content
        echo "🆕 New"
    else
        # Default to idle if we can't determine
        echo "⏸ Idle"
    end
end

function __agents_detect_opencode_state -a pane_id -d "Detect OpenCode agent state from pane content"
    # Capture pane content
    set -l pane_content (tmux capture-pane -p -t $pane_id -S -20 2>/dev/null)

    # OpenCode uses Bubble Tea TUI - try to detect common patterns
    if string match -q "*Running*" -- $pane_content; or string match -q "*Executing*" -- $pane_content
        echo "⚡Working"
    else if string match -q "*waiting*" -- $pane_content; or string match -q "*Approve*" -- $pane_content
        echo "❗Input"
    else if string match -q "*What can I help*" -- $pane_content; or string match -q "*Enter your*" -- $pane_content
        echo "⏸ Idle"
    else
        # Default to idle
        echo "⏸ Idle"
    end
end

function __agents_colorize_state -a state -d "Add ANSI color codes to state for visual emphasis"
    switch $state
        case "⚡Working"
            # Yellow for working
            echo "\033[33m$state\033[0m"
        case "❗Input"
            # Bright red/magenta for input (needs attention!)
            echo "\033[1;35m$state\033[0m"
        case "⏸ Idle"
            # Green for idle (ready)
            echo "\033[32m$state\033[0m"
        case "🆕 New"
            # Cyan for new
            echo "\033[36m$state\033[0m"
        case '*'
            # No color for unknown
            echo $state
    end
end
